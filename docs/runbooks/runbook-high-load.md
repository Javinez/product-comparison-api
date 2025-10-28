# Runbook: Manejo de Alta Carga en Product Comparison API

## Información General

**Servicio**: Product Comparison API  
**Criticidad**: Alta  
**Equipo Responsable**: Platform Team  
**Última Actualización**: 2024-10-27  
**Versión**: 1.0

## Descripción del Problema

Este runbook cubre el escenario cuando el servicio experimenta alta carga que puede resultar en:
- Latencia elevada (P99 > 500ms)
- Errores 503 Service Unavailable
- CPU > 80% sostenido
- Memory > 85%
- Rate limiting activado frecuentemente

## Detección

### Alertas Automatizadas

```yaml
Alerta: HighLatency
Condición: p99_latency > 500ms por 5 minutos
Severidad: Warning

Alerta: ServiceOverload
Condición: error_rate > 1% AND cpu > 80%
Severidad: Critical

Alerta: MemoryPressure
Condición: memory_usage > 85% por 3 minutos
Severidad: Warning
```

### Síntomas en Dashboard

- 📊 Grafana Dashboard: `Product-API-Performance`
- 📈 Métricas clave a revisar:
  - Request rate (requests/second)
  - Error rate (4xx, 5xx)
  - Latency percentiles (P50, P95, P99)
  - Active connections
  - CPU y Memory usage
  - Cache hit ratio

## Diagnóstico Rápido

### Paso 1: Verificar Estado General

```bash
# Verificar pods en ECS/K8s
kubectl get pods -n production | grep product-api
aws ecs list-tasks --cluster product-api-cluster --service-name product-api-service

# Verificar logs recientes
kubectl logs -n production -l app=product-api --tail=100
aws logs tail /ecs/product-api --follow

# Verificar métricas actuales
curl http://product-api.internal:8080/metrics | grep -E "(request_duration|request_count|error_rate)"
```

### Paso 2: Identificar Tipo de Carga

```bash
# Analizar patrones de tráfico
aws cloudwatch get-metric-statistics \
  --namespace AWS/ApplicationELB \
  --metric-name RequestCount \
  --dimensions Name=LoadBalancer,Value=app/product-api-alb/* \
  --start-time $(date -u -d '30 minutes ago' +%Y-%m-%dT%H:%M:%S) \
  --end-time $(date -u +%Y-%m-%dT%H:%M:%S) \
  --period 60 \
  --statistics Sum

# Verificar top endpoints con alta carga
grep "GET\|POST" /var/log/product-api/access.log | \
  awk '{print $7}' | sort | uniq -c | sort -rn | head -20

# Identificar IPs con más requests
awk '{print $1}' /var/log/product-api/access.log | \
  sort | uniq -c | sort -rn | head -10
```

### Paso 3: Verificar Recursos

```bash
# CPU y Memory por pod
kubectl top pods -n production | grep product-api

# Verificar límites vs uso actual
kubectl describe pod <pod-name> -n production | grep -A 5 "Limits\|Requests"

# Database connections
echo "SELECT count(*) FROM pg_stat_activity WHERE application_name = 'product-api';" | \
  psql -h $DB_HOST -U $DB_USER -d products

# Redis memory y connections
redis-cli -h $REDIS_HOST INFO memory
redis-cli -h $REDIS_HOST CLIENT LIST | wc -l
```

## Acciones de Mitigación

### 🚨 Respuesta Inmediata (< 5 minutos)

#### 1. Escalar Horizontalmente

```bash
# ECS - Aumentar desired count
aws ecs update-service \
  --cluster product-api-cluster \
  --service product-api-service \
  --desired-count 20

# Kubernetes - Escalar replicas
kubectl scale deployment product-api -n production --replicas=20

# Verificar escalado
watch "kubectl get pods -n production | grep product-api"
```

#### 2. Activar Circuit Breaker

```bash
# Activar circuit breaker para endpoints no críticos
curl -X POST http://product-api.internal:8080/admin/circuit-breaker \
  -H "Content-Type: application/json" \
  -d '{"endpoints": ["/api/v1/products/search"], "enabled": true}'

# Verificar estado
curl http://product-api.internal:8080/admin/circuit-breaker/status
```

#### 3. Aumentar Rate Limits Temporalmente

```bash
# Modificar rate limits en API Gateway
aws apigateway update-stage \
  --rest-api-id $API_ID \
  --stage-name production \
  --patch-operations \
    op=replace,path=/throttle/rateLimit,value=20000 \
    op=replace,path=/throttle/burstLimit,value=10000
```

### ⚡ Respuesta Rápida (5-15 minutos)

#### 4. Habilitar Cache Agresivo

```bash
# Aumentar TTL de cache
redis-cli -h $REDIS_HOST
> CONFIG SET maxmemory-policy allkeys-lru
> FLUSHDB  # ⚠️ Solo si el cache está corrupto

# Modificar configuración de la aplicación
kubectl set env deployment/product-api \
  CACHE_TTL=600 \
  CACHE_ENABLED=true \
  -n production
```

#### 5. Degradación Gradual de Servicio

```javascript
// Feature flags para deshabilitar features no críticas
const degradationConfig = {
  "disable_recommendations": true,
  "disable_complex_comparisons": true,
  "max_comparison_products": 3,
  "disable_search_suggestions": true,
  "cache_only_mode": false  // Activar solo si DB está sobrecargada
};

// Aplicar via API
curl -X PUT http://product-api.internal:8080/admin/features \
  -H "Content-Type: application/json" \
  -d '${JSON.stringify(degradationConfig)}'
```

#### 6. Desviar Tráfico a Región de Respaldo

```bash
# Modificar weights en Route53
aws route53 change-resource-record-sets \
  --hosted-zone-id $ZONE_ID \
  --change-batch '{
    "Changes": [{
      "Action": "UPSERT",
      "ResourceRecordSet": {
        "Name": "api.product-comparison.com",
        "Type": "A",
        "SetIdentifier": "Primary",
        "Weight": 30,
        "AliasTarget": {
          "HostedZoneId": "Z35SXDOTRQ7X7K",
          "DNSName": "us-east-1-alb.amazonaws.com",
          "EvaluateTargetHealth": true
        }
      }
    }]
  }'
```

### 🛠️ Respuesta Extendida (15-30 minutos)

#### 7. Análisis de Queries Lentos

```sql
-- Identificar queries problemáticos en PostgreSQL
SELECT 
    query,
    calls,
    mean_exec_time,
    max_exec_time,
    total_exec_time
FROM pg_stat_statements
WHERE query LIKE '%products%'
ORDER BY mean_exec_time DESC
LIMIT 10;

-- Crear índices de emergencia si es necesario
CREATE INDEX CONCURRENTLY idx_products_category_rating_emergency 
ON products(category, rating DESC) 
WHERE active = true;
```

#### 8. Optimización de JVM/Runtime

```bash
# Para servicios Java (si aplica)
export JAVA_OPTS="-Xmx2g -Xms2g -XX:+UseG1GC -XX:MaxGCPauseMillis=100"

# Para Go - Ajustar GOGC y GOMEMLIMIT
kubectl set env deployment/product-api \
  GOGC=50 \
  GOMEMLIMIT=1GiB \
  -n production
```

## Escalación

### Cuándo Escalar

Escalar al siguiente nivel si:
- Las acciones no mejoran la situación en 15 minutos
- Múltiples servicios están afectados
- Pérdida de datos es inminente
- Clientes VIP/Enterprise están impactados

### Cadena de Escalación

1. **L1 - On-Call Engineer**: Primeros 15 minutos
2. **L2 - Senior Platform Engineer**: 15-30 minutos
3. **L3 - Platform Lead**: 30-60 minutos
4. **L4 - CTO/VP Engineering**: > 60 minutos o impacto en revenue

### Contactos

| Rol | Nombre | Teléfono | Slack | Email |
|-----|--------|----------|-------|-------|
| L1 On-Call | Rotativo | Ver PagerDuty | #oncall | oncall@company.com |
| Platform Lead | John Smith | +1-555-0100 | @jsmith | jsmith@company.com |
| Database Admin | Jane Doe | +1-555-0101 | @jdoe | jdoe@company.com |
| CTO | Bob Wilson | +1-555-0102 | @bwilson | bwilson@company.com |

## Post-Mortem

### Información a Recolectar

Durante el incidente, recolectar:

```bash
# Crear directorio para evidencia
mkdir -p /tmp/incident-$(date +%Y%m%d-%H%M%S)
cd /tmp/incident-*

# Capturar métricas
curl http://product-api.internal:8080/metrics > metrics.txt
kubectl get events -n production > k8s-events.txt
kubectl describe pods -n production > pods-description.txt

# Logs de los últimos 30 minutos
kubectl logs -n production -l app=product-api --since=30m > app-logs.txt

# Estado de la base de datos
echo "SELECT * FROM pg_stat_activity;" | psql -h $DB_HOST > db-activity.txt
echo "SELECT * FROM pg_stat_statements ORDER BY total_exec_time DESC LIMIT 20;" | \
  psql -h $DB_HOST > slow-queries.txt

# Snapshot de CloudWatch metrics
aws cloudwatch get-metric-statistics \
  --namespace AWS/ECS \
  --metric-name CPUUtilization \
  --dimensions Name=ServiceName,Value=product-api-service \
  --start-time $(date -u -d '1 hour ago' +%Y-%m-%dT%H:%M:%S) \
  --end-time $(date -u +%Y-%m-%dT%H:%M:%S) \
  --period 300 \
  --statistics Average,Maximum > cpu-metrics.json

# Comprimir evidencia
tar -czf incident-evidence.tar.gz *
```

### Template de Post-Mortem

```markdown
# Post-Mortem: [Fecha] - Alta Carga en Product API

## Resumen
- **Duración**: [Inicio] - [Fin]
- **Impacto**: [Usuarios afectados, pérdida estimada]
- **Severidad**: [P1/P2/P3]

## Timeline
- HH:MM - Primera alerta recibida
- HH:MM - Diagnóstico inicial
- HH:MM - Primera acción de mitigación
- HH:MM - Servicio estabilizado
- HH:MM - Incidente resuelto

## Causa Raíz
[Descripción detallada]

## Lecciones Aprendidas
1. What went well
2. What went wrong
3. Where we got lucky

## Action Items
- [ ] Tarea 1 - Owner - Due date
- [ ] Tarea 2 - Owner - Due date
```

## Prevención

### Mejoras Recomendadas

1. **Auto-scaling más agresivo**
```yaml
scalingPolicy:
  targetCPU: 60  # Reducir de 70%
  scaleUpRate: 100%  # Duplicar capacidad
  scaleDownRate: 10%  # Reducir lentamente
```

2. **Cache warming**
```bash
# Ejecutar cada 30 minutos
*/30 * * * * /usr/local/bin/warm-cache.sh
```

3. **Load testing regular**
```bash
# Ejecutar weekly en staging
k6 run --vus 1000 --duration 30m load-test.js
```

4. **Capacity planning**
- Revisar métricas mensualmente
- Proyectar crecimiento trimestral
- Pre-escalar antes de eventos conocidos

## Referencias

- [AWS ECS Scaling Documentation](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/service-auto-scaling.html)
- [Kubernetes HPA Documentation](https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/)
- [Site Reliability Engineering Book](https://sre.google/sre-book/table-of-contents/)
- [Incident Response Playbook](internal-wiki/incident-response)

## Historial de Cambios

| Versión | Fecha | Autor | Cambios |
|---------|-------|-------|---------|
| 1.0 | 2024-10-27 | Platform Team | Versión inicial |

---

**Nota**: Este runbook debe ser revisado y actualizado después de cada incidente mayor.