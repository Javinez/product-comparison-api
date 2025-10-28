# Runbook: Debugging y Troubleshooting - Product Comparison API

## Información General

**Servicio**: Product Comparison API  
**Equipo Responsable**: Platform Team  
**Última Actualización**: 2024-10-27  
**Versión**: 1.0

## Guía Rápida de Problemas Comunes

| Síntoma | Causa Probable | Acción Inmediata | Sección |
|---------|---------------|------------------|---------|
| 503 Service Unavailable | Pods crasheando | `kubectl get pods` | [#1](#1-servicio-no-disponible-503) |
| Alta latencia | Cache miss alto | Verificar Redis | [#2](#2-alta-latencia) |
| 404 en endpoints válidos | Routing incorrecto | Check ingress rules | [#3](#3-errores-404) |
| Datos inconsistentes | Replication lag | Check DB sync | [#4](#4-inconsistencia-de-datos) |
| Memory leaks | Goroutine leaks | pprof analysis | [#5](#5-memory-leaks) |
| CPU 100% | Infinite loops | Stack traces | [#6](#6-alto-consumo-cpu) |

## Herramientas de Debugging

### Setup Inicial

```bash
# Configurar kubectl context
kubectl config use-context production

# Alias útiles
alias k='kubectl'
alias kp='kubectl get pods'
alias kl='kubectl logs'
alias ke='kubectl exec -it'
alias kd='kubectl describe'

# Herramientas necesarias
go install github.com/rakyll/hey@latest  # Load testing
go install github.com/google/pprof@latest  # Profiling
brew install k6 stern jq  # MacOS
apt-get install curl jq netcat -y  # Linux
```

## Problemas y Soluciones

### 1. Servicio No Disponible (503)

#### Diagnóstico

```bash
# 1. Verificar pods
kubectl get pods -n production | grep product-api
# Output esperado: Running 1/1

# 2. Si hay pods en CrashLoopBackOff
kubectl describe pod <pod-name> -n production | grep -A 10 "Events:"

# 3. Verificar logs del pod
kubectl logs <pod-name> -n production --previous  # Para ver logs antes del crash

# 4. Verificar resources
kubectl top pods -n production | grep product-api
```

#### Causas Comunes y Soluciones

##### A. Out of Memory (OOM)

```bash
# Verificar OOM kills
kubectl get events -n production | grep OOMKilled

# Solución temporal: Aumentar memoria
kubectl set resources deployment/product-api \
  --limits=memory=2Gi --requests=memory=1Gi \
  -n production

# Solución permanente: Encontrar memory leak
kubectl exec -it <pod-name> -n production -- \
  curl http://localhost:8080/debug/pprof/heap > heap.prof
go tool pprof heap.prof
```

##### B. Liveness Probe Failing

```bash
# Verificar configuración del probe
kubectl get deployment product-api -n production -o yaml | grep -A 10 livenessProbe

# Aumentar timeout temporalmente
kubectl patch deployment product-api -n production -p \
  '{"spec":{"template":{"spec":{"containers":[{"name":"product-api","livenessProbe":{"timeoutSeconds":10}}]}}}}'

# Verificar endpoint de health
kubectl exec -it <pod-name> -n production -- curl http://localhost:8080/health
```

##### C. Dependencias No Disponibles

```bash
# Verificar conectividad a base de datos
kubectl exec -it <pod-name> -n production -- \
  nc -zv product-db.cluster.local 5432

# Verificar Redis
kubectl exec -it <pod-name> -n production -- \
  redis-cli -h redis.cluster.local ping

# Verificar DNS
kubectl exec -it <pod-name> -n production -- nslookup product-db.cluster.local
```

### 2. Alta Latencia

#### Diagnóstico

```bash
# 1. Medir latencia actual
hey -n 1000 -c 10 https://api.product-comparison.com/api/v1/products | grep "Latency distribution"

# 2. Verificar métricas de cache
redis-cli -h redis.cluster.local
> INFO stats
> MONITOR  # Ver comandos en tiempo real (cuidado en producción!)

# 3. Trace específico de request
curl -H "X-Trace-Id: debug-$(date +%s)" https://api.product-comparison.com/api/v1/products/compare?ids=prod-001,prod-002
# Luego buscar en logs
kubectl logs -n production -l app=product-api | grep "debug-"
```

#### Optimizaciones

##### A. Cache Tuning

```bash
# Verificar hit rate
redis-cli INFO stats | grep keyspace_hits
# Si hit rate < 80%, revisar TTL y keys

# Aumentar TTL
kubectl set env deployment/product-api CACHE_TTL=600 -n production

# Pre-warm cache
for id in {001..100}; do
  curl "https://api.product-comparison.com/api/v1/products/prod-$id" &
done
```

##### B. Database Query Optimization

```sql
-- Encontrar queries lentos
SELECT query, mean_exec_time, calls 
FROM pg_stat_statements 
WHERE mean_exec_time > 100 
ORDER BY mean_exec_time DESC 
LIMIT 10;

-- Añadir índices necesarios
EXPLAIN ANALYZE 
SELECT * FROM products 
WHERE category = 'Laptops' AND rating > 4;

-- Si hay table scan, crear índice
CREATE INDEX CONCURRENTLY idx_category_rating 
ON products(category, rating);
```

##### C. Connection Pool Tuning

```go
// Verificar configuración actual
kubectl exec -it <pod-name> -n production -- env | grep -E "(POOL|CONNECTION)"

// Ajustar pool size
kubectl set env deployment/product-api \
  DB_MAX_CONNECTIONS=50 \
  DB_MAX_IDLE=10 \
  REDIS_POOL_SIZE=100 \
  -n production
```

### 3. Errores 404

#### Diagnóstico

```bash
# 1. Verificar ingress rules
kubectl get ingress -n production -o yaml

# 2. Test directo al pod
kubectl port-forward <pod-name> 8080:8080 -n production
curl http://localhost:8080/api/v1/products  # Should work

# 3. Verificar service
kubectl get svc product-api -n production
kubectl get endpoints product-api -n production
```

#### Fixes Comunes

```yaml
# Fix ingress path
kubectl edit ingress product-api -n production
# Cambiar:
#   path: /api/v1/products/*
# Por:
#   path: /api/v1/*
#   pathType: Prefix
```

### 4. Inconsistencia de Datos

#### Diagnóstico

```bash
# 1. Verificar replication lag
psql -h product-db-replica -U admin -c \
  "SELECT now() - pg_last_xact_replay_timestamp() AS replication_lag;"

# 2. Comparar datos entre instances
PRIMARY=$(psql -h product-db-primary -U admin -c "SELECT COUNT(*) FROM products;" -t)
REPLICA=$(psql -h product-db-replica -U admin -c "SELECT COUNT(*) FROM products;" -t)
echo "Primary: $PRIMARY, Replica: $REPLICA, Diff: $((PRIMARY-REPLICA))"

# 3. Verificar cache stale
redis-cli --scan --pattern "product:*" | while read key; do
  redis-cli TTL "$key"
done | sort | uniq -c
```

#### Soluciones

```bash
# Flush cache si hay inconsistencias
redis-cli FLUSHDB

# Forzar sincronización
psql -h product-db-replica -U admin -c "SELECT pg_wal_replay_resume();"

# Invalidar cache específico
kubectl exec -it <pod-name> -n production -- \
  curl -X DELETE http://localhost:8080/admin/cache/product:prod-001
```

### 5. Memory Leaks

#### Diagnóstico con pprof

```bash
# 1. Habilitar pprof endpoint (si no está habilitado)
kubectl port-forward <pod-name> 6060:6060 -n production

# 2. Capturar heap profile
curl http://localhost:6060/debug/pprof/heap > heap.prof

# 3. Analizar
go tool pprof -http=:8081 heap.prof
# Abrir browser en http://localhost:8081

# 4. Comparar snapshots
curl http://localhost:6060/debug/pprof/heap > heap1.prof
sleep 300  # Esperar 5 minutos
curl http://localhost:6060/debug/pprof/heap > heap2.prof
go tool pprof -diff_base=heap1.prof heap2.prof
```

#### Análisis de Goroutines

```bash
# Ver número de goroutines
curl http://localhost:6060/debug/pprof/goroutine?debug=1 | head -20

# Si hay miles de goroutines, buscar leaks
curl http://localhost:6060/debug/pprof/goroutine?debug=2 > goroutines.txt
grep "goroutine [0-9]" goroutines.txt | wc -l

# Identificar goroutines stuck
grep -A 5 "minutes" goroutines.txt
```

#### Fix Común: Context Cancellation

```go
// Código problemático
func (h *Handler) SlowOperation(w http.ResponseWriter, r *http.Request) {
    go func() {
        // Esta goroutine nunca termina si el cliente cancela
        time.Sleep(10 * time.Minute)
        processData()
    }()
}

// Código corregido
func (h *Handler) SlowOperation(w http.ResponseWriter, r *http.Request) {
    go func() {
        select {
        case <-time.After(10 * time.Minute):
            processData()
        case <-r.Context().Done():
            return // Cliente canceló, terminar goroutine
        }
    }()
}
```

### 6. Alto Consumo CPU

#### Diagnóstico

```bash
# 1. CPU profile
curl http://localhost:6060/debug/pprof/profile?seconds=30 > cpu.prof
go tool pprof -http=:8082 cpu.prof

# 2. Trace execution
curl http://localhost:6060/debug/pprof/trace?seconds=5 > trace.out
go tool trace trace.out

# 3. Live monitoring
kubectl exec -it <pod-name> -n production -- top -H -p 1
```

#### Problemas Comunes

##### A. Infinite Loop

```go
// Buscar en código
grep -r "for {" --include="*.go" .
grep -r "for true" --include="*.go" .

// Añadir circuit breaker
for i := 0; i < maxIterations; i++ {
    if condition {
        break
    }
    // ... work
}
```

##### B. Regex Costoso

```go
// Problema: Regex compilado en cada request
func validateInput(input string) bool {
    re := regexp.MustCompile(`^[a-zA-Z0-9]+$`)  // Costoso!
    return re.MatchString(input)
}

// Solución: Compilar una vez
var validationRegex = regexp.MustCompile(`^[a-zA-Z0-9]+$`)
func validateInput(input string) bool {
    return validationRegex.MatchString(input)
}
```

## Comandos de Emergencia

### Kill Switch - Deshabilitar Features

```bash
# Deshabilitar feature problemático
kubectl set env deployment/product-api \
  FEATURE_COMPARISON=false \
  FEATURE_SEARCH=false \
  -n production

# Modo solo-lectura
kubectl set env deployment/product-api \
  READONLY_MODE=true \
  -n production
```

### Restart Rápido

```bash
# Restart graceful
kubectl rollout restart deployment/product-api -n production

# Force restart (último recurso)
kubectl delete pods -n production -l app=product-api
```

### Rollback Inmediato

```bash
# Ver revisiones
kubectl rollout history deployment/product-api -n production

# Rollback a versión anterior
kubectl rollout undo deployment/product-api -n production

# Rollback a versión específica
kubectl rollout undo deployment/product-api --to-revision=42 -n production
```

## Scripts de Debugging

### debug-pod.sh

```bash
#!/bin/bash
POD=$1
NAMESPACE=${2:-production}

echo "=== Pod Debugging Info ==="
echo "Pod: $POD"
echo "Namespace: $NAMESPACE"
echo ""

echo "=== Status ==="
kubectl get pod $POD -n $NAMESPACE

echo -e "\n=== Resources ==="
kubectl top pod $POD -n $NAMESPACE

echo -e "\n=== Recent Events ==="
kubectl get events -n $NAMESPACE --field-selector involvedObject.name=$POD

echo -e "\n=== Environment ==="
kubectl exec $POD -n $NAMESPACE -- env | grep -E "(DATABASE|REDIS|CACHE)"

echo -e "\n=== Network Connectivity ==="
kubectl exec $POD -n $NAMESPACE -- nc -zv product-db.cluster.local 5432
kubectl exec $POD -n $NAMESPACE -- nc -zv redis.cluster.local 6379

echo -e "\n=== Last 50 Log Lines ==="
kubectl logs $POD -n $NAMESPACE --tail=50

echo -e "\n=== Health Check ==="
kubectl exec $POD -n $NAMESPACE -- curl -s http://localhost:8080/health | jq .
```

### trace-request.sh

```bash
#!/bin/bash
TRACE_ID="trace-$(date +%s)-$$"
ENDPOINT=${1:-"/api/v1/products"}

echo "Tracing request with ID: $TRACE_ID"
echo "Endpoint: $ENDPOINT"

# Hacer request con trace ID
curl -H "X-Trace-Id: $TRACE_ID" \
     -H "X-Debug: true" \
     -w "\n\nTime Total: %{time_total}s\n" \
     https://api.product-comparison.com$ENDPOINT

echo -e "\n=== Searching logs for trace $TRACE_ID ==="
sleep 2  # Dar tiempo para que los logs se propaguen

# Buscar en todos los pods
kubectl logs -n production -l app=product-api --since=1m | \
  grep "$TRACE_ID" | \
  jq -r '.timestamp + " | " + .level + " | " + .message' 2>/dev/null || \
  grep "$TRACE_ID"
```

## Métricas Clave para Monitoreo

```promql
# Latencia P99
histogram_quantile(0.99, 
  rate(http_request_duration_seconds_bucket[5m])
)

# Error rate
rate(http_requests_total{status=~"5.."}[5m]) / 
rate(http_requests_total[5m])

# Goroutines activos
go_goroutines{job="product-api"}

# Memory usage
process_resident_memory_bytes{job="product-api"} / 1024 / 1024

# Cache hit ratio
rate(cache_hits_total[5m]) / 
(rate(cache_hits_total[5m]) + rate(cache_misses_total[5m]))

# Database connections
mysql_global_status_threads_connected{job="product-db"}
```

## Referencias

- [Go Debugging Guide](https://golang.org/doc/diagnostics)
- [Kubernetes Troubleshooting](https://kubernetes.io/docs/tasks/debug/)
- [pprof Tutorial](https://jvns.ca/blog/2017/09/24/profiling-go-with-pprof/)
- [Distributed Tracing Best Practices](internal-wiki/tracing)

---

**Tip**: Mantén siempre `stern` corriendo durante debugging:
```bash
stern product-api -n production --since 1m
```