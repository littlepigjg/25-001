# 缺陷候选清单 (Bug Catalog)

## 项目概述
- **项目类型**: 实时日志聚合与告警规则引擎
- **缺陷数量**: 30
- **规模**: 大型 (50+ Go文件, 7800+ 行)

---

## 缺陷清单

| bug_id | bug_category | 缺陷描述 | 植入位置 | 预期表现 | 触发方式 | 缺陷难度 |
|--------|-------------|---------|---------|---------|---------|---------|
| logalert-concur-1 | concurrency | 日志存储GetStats与写入并发导致data race | internal/store/memory_log_store.go.Stats, internal/store/memory_log_store.go.Store | 高并发下panic "concurrent map read and map write" | 并发请求同时写入和读取日志 | ⭐⭐⭐ |
| logalert-concur-2 | concurrency | WaitGroup计数不匹配导致goroutine泄露 | internal/service/scheduler_service.go.StartAlertScan, internal/service/scheduler_service.go.Stop | 服务器关闭时goroutine无法正常退出 | 启动调度服务后立即关闭 | ⭐⭐⭐⭐ |
| logalert-concur-3 | concurrency | channel未关闭导致生产者goroutine泄露 | internal/service/alert_service.go.EvaluateRule, internal/service/scheduler_service.go.startAlertScan | 告警评估后goroutine持续运行 | 创建100条错误日志触发规则 | ⭐⭐⭐ |
| logalert-concur-4 | concurrency | 规则存储Update中map读写未加锁 | internal/store/memory_rule_store.go.Update, internal/store/memory_rule_store.go.List | 并发更新和列出规则时panic | 同时执行更新和查询规则 | ⭐⭐⭐ |
| logalert-concur-5 | concurrency | 告警事件存储索引更新非原子性 | internal/store/memory_alert_store.go.Store, internal/store/memory_alert_store.go.ListActive | 并发创建和列出告警时数据不一致 | 多协程同时触发告警 | ⭐⭐⭐⭐ |
| logalert-concur-6 | concurrency | SafeMap的keys方法与Set并发不安全性 | pkg/syncutil/safe_map.go.Keys, pkg/syncutil/safe_map.go.Set | 返回的keys列表可能包含已删除的key | 并发调用Keys和Set | ⭐⭐⭐ |
| logalert-concur-7 | concurrency | 调度服务Stop与start竞态条件 | internal/service/scheduler_service.go.Start, internal/service/scheduler_service.go.Stop | 快速重启时可能无法完全停止 | 快速连续调用Start/Stop | ⭐⭐⭐⭐ |
| logalert-nil-8 | nil | 向nil map写入metadata导致panic | internal/model/log_entry.go.WithMetadata, internal/service/log_service.go.CreateLog | 调用WithMetadata时panic "assignment to entry in nil map" | 创建日志时调用WithMetadata | ⭐⭐ |
| logalert-nil-9 | nil | GetStoreStats返回nil时直接访问字段 | internal/service/log_service.go.GetStoreStats, internal/handler/health_handler.go.Healthy | panic "nil pointer dereference" | 存储为空时访问健康检查 | ⭐⭐ |
| logalert-nil-10 | nil | nil日志条目的Source字段访问未校验 | internal/service/stats_service.go.GetStatistics, internal/store/memory_log_store.go.Query | panic "nil pointer dereference" | 查询返回nil entry | ⭐⭐⭐ |
| logalert-nil-11 | nil | SafeMap.Get返回零值与nil混淆 | pkg/syncutil/safe_map.go.Get, internal/store/memory_log_store.go.evictOldest | 返回零值导致逻辑错误 | 存储空时调用Get | ⭐⭐⭐ |
| logalert-nil-12 | nil | AlertEvent的ResolvedAt为nil时直接格式化 | internal/model/alert_event.go.Duration, internal/handler/alert_handler.go.ListAlerts | panic "nil pointer dereference" | 查看未解决告警的Duration | ⭐⭐ |
| logalert-slice-13 | slice | append到slice后底层数组共享导致数据污染 | internal/store/memory_log_store.go.Query, internal/service/log_service.go.QueryLogs | 修改返回结果影响存储数据 | 查询日志后修改items | ⭐⭐⭐ |
| logalert-slice-14 | slice | 子切片写回污染原数组导致状态错乱 | internal/store/memory_log_store.go.removeEntry, internal/store/memory_log_store.go.evictOldest | 淘汰日志时可能丢弃错误的条目 | 存储满时触发淘汰 | ⭐⭐⭐⭐ |
| logalert-slice-15 | slice | 容量估算不足导致slice频繁扩容 | internal/store/memory_log_store.go.Store, internal/config/config.go.DefaultConfig | 高频写入时性能下降 | 每秒1000条日志持续写入 | ⭐⭐ |
| logalert-slice-16 | slice | 分页slice边界条件处理错误 | internal/store/memory_log_store.go.Query, internal/model/query.go.Validate | offset+limit超出范围时panic | offset > total的查询 | ⭐⭐⭐ |
| logalert-slice-17 | slice | Sort.Slice比较函数不稳定排序 | internal/store/memory_alert_store.go.ListActive, internal/store/memory_alert_store.go.sortEventsByTime | 告警列表排序不稳定 | 并发触发多个告警后列出 | ⭐⭐⭐⭐ |
| logalert-error-18 | error | %w丢失导致errors.Is/As失效 | pkg/errors/errors.go.Wrap, internal/service/log_service.go.CreateLog | errors.Is无法匹配包装后的错误 | 使用errors.Is检查类型 | ⭐⭐⭐⭐ |
| logalert-error-19 | error | err被:=遮蔽导致原始错误丢失 | internal/service/alert_service.go.EvaluateRule, internal/store/memory_log_store.go.Query | 错误处理逻辑失效 | 查询日志时出错 | ⭐⭐⭐ |
| logalert-error-20 | error | 只比较错误字符串而非类型 | internal/service/rule_service.go.CreateRule, internal/handler/rule_handler.go.CreateRule | 自定义错误被误判 | 规则名重复时创建 | ⭐⭐⭐ |
| logalert-error-21 | error | JSON解析错误未传递导致静默失败 | pkg/jsonutil/json.go.Decode, internal/handler/log_handler.go.CreateLog | 请求失败但无错误响应 | 发送畸形JSON请求 | ⭐⭐ |
| logalert-error-22 | error | 错误码映射硬编码与实际不符 | pkg/errors/codes.go.GetMessage, internal/handler/health_handler.go.Healthy | 返回错误码对应错误消息不正确 | 触发400/404等错误 | ⭐⭐⭐ |
| logalert-context-23 | context | context取消未传播到子goroutine | internal/service/scheduler_service.go.Start, internal/service/scheduler_service.go.Stop | 服务器关闭时子任务继续运行 | SIGTERM后等待30秒以上 | ⭐⭐⭐⭐ |
| logalert-context-24 | context | 将context存入结构体后复用导致过期 | internal/service/scheduler_service.go.scanAlerts, internal/service/alert_service.go.EvaluateRule | 长时间运行后context已取消 | 运行超过1分钟的扫描任务 | ⭐⭐⭐⭐⭐ |
| logalert-context-25 | context | 忽略ctx.Err()导致取消不响应 | internal/service/stats_service.go.GetStatistics, internal/service/stats_service.go.GetHourlyTrends | 取消请求后仍继续处理 | 请求超时后服务器仍处理 | ⭐⭐⭐ |
| logalert-context-26 | context | HTTP超时中间件context传递遗漏 | pkg/httputil/middleware.go.TimeoutMiddleware, internal/handler/log_handler.go.CreateLog | 超时后日志仍被写入 | 发送大型日志请求并设置超时 | ⭐⭐⭐⭐ |
| logalert-defer-27 | defer | 循环内defer直到函数返回才执行 | internal/service/log_service.go.CreateBatchLogs, internal/store/memory_log_store.go.Store | 批量存储时文件描述符泄露 | 批量创建1000条日志 | ⭐⭐⭐⭐ |
| logalert-defer-28 | defer | error分支跳过了资源释放 | internal/service/cleanup_service.go.CleanupExpiredLogs, internal/store/memory_log_store.go.Cleanup | 清理失败时资源未正确释放 | 清理过期日志时发生错误 | ⭐⭐⭐ |
| logalert-defer-29 | defer | defer修改命名返回值导致逻辑错误 | internal/service/window_manager.go.ValidateWindow, internal/handler/query_handler.go.SearchLogs | 返回错误被defer修改 | 查询包含非法时间范围 | ⭐⭐⭐⭐ |
| logalert-other-30 | other | 时间窗口计算使用UTC而非本地时区 | internal/service/alert_service.go.EvaluateRule, internal/service/window_manager.go.GetAlertWindow | 告警触发时间偏差8小时 | 非UTC时区部署 | ⭐⭐⭐⭐ |

---

## 缺陷分布统计

| 缺陷类别 | 数量 | 占比 |
|---------|------|------|
| concurrency | 7 | 23.3% |
| nil | 5 | 16.7% |
| slice | 5 | 16.7% |
| error | 5 | 16.7% |
| context | 4 | 13.3% |
| defer | 3 | 10.0% |
| other | 1 | 3.3% |
| **合计** | **30** | **100%** |

## 单文件缺陷占比
- 每个 .go 文件中缺陷数均 ≤ 30%（即9个缺陷）
- 存储层文件分布: 6个缺陷
- 服务层文件分布: 12个缺陷
- 处理器层文件分布: 4个缺陷
- 工具包层文件分布: 8个缺陷

## 验证命令

```bash
# 编译所有代码
go build ./...

# 静态分析
go vet ./...

# 运行带竞态检测的测试
go test -race -count=1 ./...

# 检查代码覆盖率
go test -cover -count=1 ./...
```
