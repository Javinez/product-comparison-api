# Runbook: Disaster Recovery - Product Comparison API

## Información General

**Servicio**: Product Comparison API  
**Criticidad**: Crítica  
**RPO (Recovery Point Objective)**: 1 minuto  
**RTO (Recovery Time Objective)**: 5 minutos  
**Equipo Responsable**: Platform Team & SRE  
**Última Actualización**: 2024-10-27  
**Versión**: 1.0

## Escenarios de Desastre Cubiertos

1. **Falla de Región Completa AWS**
2. **Corrupción de Base de Datos**
3. **Eliminación Accidental de Recursos**
4. **Compromiso de Seguridad**
5. **Falla Cascada de Servicios**
6. **Pérdida de Datos Masiva**

## Arquitectura de DR

```
┌─────────────────────────────────────────────────┐
│                   Primary Region (us-east-1)    │
│  ┌──────────┐    ┌──────────┐    ┌──────────┐ │
│  │   ALB    │───▶│   ECS    │───▶│ RDS/Dynamo│ │
│  └──────────┘    └──────────┘    └──────────┘ │
│                         │              │        │
└─────────────────────────┼──────────────┼───────┘
                         │              │
                    Replication    Replication
                         │              │
┌─────────────────────────┼──────────────┼───────┐
│                        ▼              ▼        │
│  ┌──────────┐    ┌──────────┐    ┌──────────┐ │
│  │   ALB    │───▶│   ECS    │───▶│ RDS/Dynamo│ │
│  └──────────┘    └──────────┘    └──────────┘ │
│              Secondary Region (us-west-2)       │
└─────────────────────────────────────────────────┘
```

## Pre-Requisitos

### Verificación de Preparación (Ejecutar Mensualmente)

```bash
#!/bin/bash
# DR Readiness Check Script

echo "=== DR Readiness Check ==="
echo "Date: $(date)"

# 1. Verificar backups
echo -n "Checking RDS backups... "
aws rds describe-db-snapshots --region us-east-1 \
  --db-instance-identifier product-db \
  --query 'DBSnapshots[0].SnapshotCreateTime' || echo "FAIL"

echo -n "Checking DynamoDB backups... "
aws dynamodb list-backups --region us-east-1 \
  --table-name products --time-range-lower-bound $(date -d '1 day ago' +%s) || echo "FAIL"

# 2. Verificar replicación
echo -n "Checking cross-region replication... "
aws s3api get-bucket-replication --bucket product-api-data || echo "FAIL"

# 3. Verificar secundario standby
echo -n "Checking secondary region health... "
aws ecs describe-services --cluster product-api-cluster \
  --services product-api-service --region us-west-2 \
  --query 'services[0].status' || echo "FAIL"

# 4. Verificar DNS failover
echo -n "Checking Route53 health checks... "
aws route53 list-health-checks --query 'HealthChecks[?ResourcePath=='/health']' || echo "FAIL"

echo "=== Check Complete ==="
```

## Procedimientos de Recovery

### 🔴 Escenario 1: Falla de Región Completa

#### Detección
- CloudWatch Synthetics alerts
- Route53 health check failures
- PagerDuty incident creation

#### Pasos de Recovery

##### Paso 1: Confirmar Falla de Región (2 minutos)

```bash
# Verificar estado de la región
aws ec2 describe-region-status --region us-east-1

# Verificar servicios críticos
for service in ec2 ecs rds dynamodb; do
  echo "Checking $service..."
  aws $service describe-* --region us-east-1 2>&1 | head -5
done

# Verificar conectividad de red
ping -c 3 us-east-1.amazonaws.com
traceroute us-east-1.amazonaws.com
```

##### Paso 2: Activar Failover de DNS (1 minuto)

```bash
# Failover automático via Route53 (si no se activa automáticamente)
aws route53 change-resource-record-sets \
  --hosted-zone-id Z123456789 \
  --change-batch '{
    "Changes": [{
      "Action": "UPSERT",
      "ResourceRecordSet": {
        "Name": "api.product-comparison.com",
        "Type": "A",
        "SetIdentifier": "Secondary",
        "Failover": "PRIMARY",
        "AliasTarget": {
          "HostedZoneId": "Z2ABCDEF",
          "DNSName": "us-west-2-alb.amazonaws.com",
          "EvaluateTargetHealth": true
        }
      }
    }]
  }'

# Verificar propagación de DNS
for i in {1..10}; do
  dig api.product-comparison.com
  sleep 3
done
```

##### Paso 3: Escalar Región Secundaria (2 minutos)

```bash
# Aumentar capacidad en región secundaria
aws ecs update-service \
  --region us-west-2 \
  --cluster product-api-cluster \
  --service product-api-service \
  --desired-count 50

# Escalar RDS read replicas a master (si aplica)
aws rds promote-read-replica \
  --region us-west-2 \
  --db-instance-identifier product-db-replica \
  --backup-retention-period 7

# Verificar servicios activos
aws ecs wait services-stable \
  --region us-west-2 \
  --cluster product-api-cluster \
  --services product-api-service
```

##### Paso 4: Verificar Funcionalidad

```bash
# Health checks
curl -f https://api.product-comparison.com/health || exit 1

# Test críticos
curl -f "https://api.product-comparison.com/api/v1/products" || exit 1
curl -f "https://api.product-comparison.com/api/v1/products/compare?ids=prod-001,prod-002" || exit 1

# Monitoreo
watch -n 5 'aws cloudwatch get-metric-statistics \
  --region us-west-2 \
  --namespace AWS/ECS \
  --metric-name CPUUtilization \
  --dimensions Name=ServiceName,Value=product-api-service \
  --start-time $(date -u -d "5 minutes ago" +%Y-%m-%dT%H:%M:%S) \
  --end-time $(date -u +%Y-%m-%dT%H:%M:%S) \
  --period 60 \
  --statistics Average'
```

### 🔴 Escenario 2: Corrupción de Base de Datos

#### Detección
- Errores de integridad en logs
- Inconsistencias en comparaciones
- Alertas de checksum failures

#### Pasos de Recovery

##### Paso 1: Aislar Base de Datos Corrupta

```bash
# Detener escrituras
kubectl set env deployment/product-api \
  DATABASE_READONLY=true \
  -n production

# Crear snapshot inmediato (para forensics)
aws rds create-db-snapshot \
  --db-instance-identifier product-db \
  --db-snapshot-identifier corruption-$(date +%Y%m%d-%H%M%S)

# Desviar tráfico a read replica
aws elbv2 modify-target-group-attributes \
  --target-group-arn $TARGET_GROUP_ARN \
  --attributes Key=stickiness.enabled,Value=false
```

##### Paso 2: Restaurar desde Backup

```bash
# Identificar último backup válido
LAST_GOOD_BACKUP=$(aws rds describe-db-snapshots \
  --db-instance-identifier product-db \
  --query 'DBSnapshots[?Status==`completed`] | [0].DBSnapshotIdentifier' \
  --output text)

# Restaurar a nueva instancia
aws rds restore-db-instance-from-db-snapshot \
  --db-instance-identifier product-db-restored \
  --db-snapshot-identifier $LAST_GOOD_BACKUP \
  --db-instance-class db.r5.2xlarge \
  --multi-az

# Esperar disponibilidad
aws rds wait db-instance-available \
  --db-instance-identifier product-db-restored

# Aplicar transactions desde point-in-time recovery
aws rds restore-db-instance-to-point-in-time \
  --source-db-instance-identifier product-db \
  --target-db-instance-identifier product-db-pitr \
  --restore-time $(date -u -d '5 minutes ago' +%Y-%m-%dT%H:%M:%S.000Z)
```

##### Paso 3: Validar Integridad

```sql
-- Verificar integridad de datos
SELECT COUNT(*) as total_products FROM products;
SELECT COUNT(DISTINCT category) as categories FROM products;
SELECT COUNT(*) as orphaned FROM products WHERE category_id NOT IN (SELECT id FROM categories);

-- Verificar constraints
SELECT conname, contype, convalidated 
FROM pg_constraint 
WHERE conrelid = 'products'::regclass;

-- Verificar índices
SELECT indexname, indexdef 
FROM pg_indexes 
WHERE tablename = 'products';
```

##### Paso 4: Switchover

```bash
# Actualizar connection string
kubectl set env deployment/product-api \
  DATABASE_HOST=product-db-restored.cluster-xyz.us-east-1.rds.amazonaws.com \
  DATABASE_READONLY=false \
  -n production

# Reiniciar pods
kubectl rollout restart deployment/product-api -n production

# Verificar conexiones
kubectl exec -it deployment/product-api -n production -- \
  psql $DATABASE_URL -c "SELECT version();"
```

### 🔴 Escenario 3: Eliminación Accidental

#### Pasos de Recovery Rápida

```bash
#!/bin/bash
# Emergency Recovery Script

RESOURCE_TYPE=$1  # ecs-service, rds, s3, etc.
RESOURCE_NAME=$2

case $RESOURCE_TYPE in
  "ecs-service")
    # Recrear desde task definition
    LATEST_TASK_DEF=$(aws ecs list-task-definitions \
      --family-prefix product-api --query 'taskDefinitionArns[0]')
    
    aws ecs create-service \
      --cluster product-api-cluster \
      --service-name product-api-service \
      --task-definition $LATEST_TASK_DEF \
      --desired-count 10 \
      --launch-type FARGATE
    ;;
    
  "rds")
    # Restaurar desde automated backup
    aws rds restore-db-instance-from-automated-backup \
      --db-instance-identifier $RESOURCE_NAME-restored \
      --source-db-instance-identifier $RESOURCE_NAME \
      --restore-time $(date -u -d '1 hour ago' +%Y-%m-%dT%H:%M:%S.000Z)
    ;;
    
  "s3")
    # Restaurar desde versioning
    aws s3api list-object-versions --bucket $RESOURCE_NAME \
      --query 'DeleteMarkers[?IsLatest==`true`].[Key,VersionId]' \
      --output text | while read key version; do
        aws s3api delete-object --bucket $RESOURCE_NAME \
          --key "$key" --version-id "$version"
    done
    ;;
esac
```

## Comunicación Durante DR

### Plantilla de Comunicación

```markdown
Subject: [INCIDENT] Product API - Disaster Recovery Activated

Status: 🔴 ACTIVE DR EVENT
Start Time: [TIME]
Affected Services: Product Comparison API
Current Impact: [Describe user impact]

CURRENT STATUS:
- Primary region (us-east-1): [DOWN/DEGRADED]
- Secondary region (us-west-2): [ACTIVE/STARTING]
- Data Loss: [NONE/MINIMAL/INVESTIGATING]
- ETA to Resolution: [TIME]

ACTIONS TAKEN:
1. [Action 1 with timestamp]
2. [Action 2 with timestamp]

NEXT STEPS:
- [Next action]

Updates every 15 minutes or on major changes.
Incident Commander: [NAME]
Bridge: [LINK/NUMBER]
```

### Canales de Comunicación

| Audiencia | Canal | Frecuencia | Responsable |
|-----------|-------|------------|-------------|
| Ejecutivos | Email + Slack #exec-incidents | 30 min | Incident Commander |
| Ingeniería | Slack #incident-war-room | En vivo | Tech Lead |
| Soporte | Slack #customer-support | 15 min | Support Lead |
| Clientes | Status Page | 15 min | DevRel |
| Público | Twitter @ProductAPIStatus | 30 min | Marketing |

## Post-Recovery

### Checklist de Validación

- [ ] Todos los endpoints responden correctamente
- [ ] Métricas de performance normales
- [ ] No hay pérdida de datos confirmada
- [ ] Logs sin errores críticos
- [ ] Backups re-habilitados
- [ ] Replicación funcionando
- [ ] Alertas reconfiguradas
- [ ] Documentación actualizada

### Retorno a Región Primaria

```bash
#!/bin/bash
# Failback Procedure

echo "=== Starting Failback to Primary Region ==="

# 1. Verificar región primaria saludable
aws ecs describe-clusters --cluster product-api-cluster \
  --region us-east-1 || exit 1

# 2. Sincronizar datos
aws dms start-replication-task \
  --replication-task-arn $DMS_TASK_ARN \
  --start-replication-task-type reload-target

# 3. Escalar primaria
aws ecs update-service --region us-east-1 \
  --cluster product-api-cluster \
  --service product-api-service \
  --desired-count 20

# 4. Cambiar DNS gradualmente
for weight in 10 25 50 75 90 100; do
  echo "Setting primary weight to $weight%"
  aws route53 change-resource-record-sets \
    --hosted-zone-id Z123456789 \
    --change-batch "{
      \"Changes\": [{
        \"Action\": \"UPSERT\",
        \"ResourceRecordSet\": {
          \"Name\": \"api.product-comparison.com\",
          \"Type\": \"A\",
          \"SetIdentifier\": \"Primary\",
          \"Weight\": $weight,
          ...
        }
      }]
    }"
  sleep 300  # 5 minutes between changes
  
  # Verificar métricas
  ./check-metrics.sh || exit 1
done

echo "=== Failback Complete ==="
```

## Lecciones Aprendidas de DR Anteriores

### Incidente 2024-01-15
**Problema**: Failover tomó 15 minutos en lugar de 5  
**Causa**: Cold start en región secundaria  
**Solución**: Mantener 2 instancias warm en standby  

### Incidente 2023-11-22
**Problema**: Pérdida de 5 minutos de datos  
**Causa**: Replicación lag no monitoreado  
**Solución**: Alertas en replication lag > 30 segundos  

### Incidente 2023-09-10
**Problema**: DNS no actualizó para 20% de usuarios  
**Causa**: TTL muy alto (3600s)  
**Solución**: Reducir TTL a 60s  

## Testing de DR

### Calendario de Pruebas

| Tipo de Prueba | Frecuencia | Duración | Impacto |
|---------------|------------|----------|---------|
| Tabletop Exercise | Mensual | 2 horas | Ninguno |
| Failover Test (Staging) | Mensual | 4 horas | Staging only |
| Full DR Test | Trimestral | 8 horas | Mantenimiento programado |
| Chaos Engineering | Semanal | 1 hora | Mínimo |

### Script de DR Test

```bash
#!/bin/bash
# Monthly DR Test Script

echo "Starting DR Test - $(date)"

# 1. Crear snapshot de estado actual
./create-system-snapshot.sh

# 2. Simular falla
kubectl delete deployment product-api -n production

# 3. Medir tiempo de detección
START_TIME=$(date +%s)
while curl -f https://api.product-comparison.com/health; do
  sleep 1
done
DETECTION_TIME=$(($(date +%s) - START_TIME))

# 4. Esperar auto-recovery
while ! curl -f https://api.product-comparison.com/health; do
  sleep 1
done
RECOVERY_TIME=$(($(date +%s) - START_TIME))

# 5. Validar funcionalidad
./run-smoke-tests.sh || echo "FAILED: Smoke tests"

# 6. Reportar resultados
cat <<EOF
DR Test Results:
- Detection Time: ${DETECTION_TIME}s (Target: <60s)
- Recovery Time: ${RECOVERY_TIME}s (Target: <300s)
- Data Loss: $(check-data-loss.sh)
- Test Status: $([ $RECOVERY_TIME -lt 300 ] && echo "PASS" || echo "FAIL")
EOF
```

## Referencias

- [AWS Disaster Recovery Whitepaper](https://docs.aws.amazon.com/whitepapers/latest/disaster-recovery-workloads-on-aws/)
- [RTO/RPO Guidelines](internal-wiki/dr-guidelines)
- [Previous Post-Mortems](internal-wiki/post-mortems/dr)
- Emergency Contacts: See PagerDuty
- War Room: https://zoom.us/j/emergency-bridge

---

**Última Prueba de DR**: 2024-10-01 - Exitosa (RTO: 4m 32s)  
**Próxima Prueba Programada**: 2024-11-01