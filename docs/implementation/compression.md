# Takhin 压缩功能实现文档

## 概述

Takhin 实现了 Kafka 兼容的消息压缩功能，支持 5 种压缩类型，为消息存储和传输提供灵活的压缩选项。

## 支持的压缩类型

### 1. None (Type = 0)
- **描述**: 无压缩，直接传递原始数据
- **用途**: 不需要压缩的场景，节省 CPU 开销
- **性能**: 最快（无计算开销）
- **压缩率**: 100%（无压缩）

### 2. GZIP (Type = 1)
- **描述**: 标准 GZIP 压缩，使用 Go 标准库 `compress/gzip`
- **用途**: 通用场景，平衡压缩率和性能
- **性能**: 
  - 压缩: ~86 µs/op
  - 解压: ~9.5 µs/op
  - 内存分配: 814 KB (压缩), 58 KB (解压)
- **压缩率**: 3.87% (测试数据 2250 → 87 字节)
- **特点**: Kafka 标准压缩算法，兼容性最好

### 3. Snappy (Type = 2)
- **描述**: Google 开发的高速压缩算法
- **依赖**: `github.com/golang/snappy v1.0.0`
- **用途**: 性能优先场景，适合实时数据流
- **性能**:
  - 压缩: ~1.2 µs/op ⭐ **最快**
  - 解压: ~1.0 µs/op ⭐ **最快**
  - 内存分配: 5 KB (压缩), 4 KB (解压)
- **压缩率**: 6.80% (测试数据 2250 → 153 字节)
- **特点**: 速度极快，适合高吞吐量场景

### 4. LZ4 (Type = 3)
- **描述**: 高速压缩算法，平衡速度和压缩率
- **依赖**: `github.com/pierrec/lz4/v4 v4.1.22`
- **用途**: 平衡场景，Kafka 常用压缩算法
- **性能**:
  - 压缩: ~1.7 µs/op
  - 解压: ~292 µs/op
  - 内存分配: 794 B (压缩), 4 MB (解压)
- **压缩率**: 4.18% (测试数据 2250 → 94 字节)
- **特点**: 压缩快，解压稍慢

### 5. ZSTD (Type = 4)
- **描述**: Facebook 开发的高压缩率算法
- **依赖**: `github.com/klauspost/compress/zstd v1.18.2`
- **用途**: 存储优先场景，需要最佳压缩率
- **性能**:
  - 压缩: ~297 µs/op
  - 解压: ~20 µs/op
  - 内存分配: 2.3 MB (压缩), 38 KB (解压)
- **压缩率**: 3.07% ⭐ **最佳** (测试数据 2250 → 69 字节)
- **特点**: 压缩率最高，适合存储密集型应用

## 性能对比总结

| 压缩类型 | 压缩速度 | 解压速度 | 压缩率 | 推荐场景 |
|---------|---------|---------|--------|---------|
| None | 0 µs ⭐ | 0 µs ⭐ | 100% | 不需要压缩 |
| Snappy | 1.2 µs ⭐⭐ | 1.0 µs ⭐⭐ | 6.80% | 实时数据流，高吞吐量 |
| LZ4 | 1.7 µs ⭐⭐ | 292 µs | 4.18% | 平衡场景，Kafka 默认 |
| GZIP | 86 µs | 9.5 µs ⭐ | 3.87% ⭐ | 通用场景，兼容性好 |
| ZSTD | 297 µs | 20 µs | 3.07% ⭐⭐ | 存储密集型，最佳压缩率 |

**选择建议**:
- **高吞吐量场景**: Snappy (最快)
- **平衡场景**: LZ4 或 GZIP
- **存储优先**: ZSTD (最佳压缩率)
- **兼容性优先**: GZIP (Kafka 标准)

## API 使用

### 基本用法

```go
import "github.com/takhin-data/takhin/pkg/compression"

// 压缩数据
data := []byte("hello world")
compressed, err := compression.Compress(compression.GZIP, data)
if err != nil {
    // 处理错误
}

// 解压数据
decompressed, err := compression.Decompress(compression.GZIP, compressed)
if err != nil {
    // 处理错误
}
```

### 压缩类型

```go
const (
    None   Type = 0  // 无压缩
    GZIP   Type = 1  // GZIP 压缩
    Snappy Type = 2  // Snappy 压缩
    LZ4    Type = 3  // LZ4 压缩
    ZSTD   Type = 4  // ZSTD 压缩
)
```

### 错误处理

```go
compressed, err := compression.Compress(compression.Type(99), data)
if err != nil {
    // 返回错误: "unsupported compression type: 99"
}
```

## 架构设计

### 设计原则

1. **简单性优先**: 使用函数式设计，而非接口抽象
2. **零拷贝**: 直接操作字节数组，避免不必要的内存分配
3. **性能优化**: 使用 buffer pool 减少内存分配
4. **错误处理**: 统一错误返回格式

### 实现细节

**文件结构**:
```
pkg/compression/
├── compression.go       # 核心实现
└── compression_test.go  # 测试和基准测试
```

**核心函数**:
```go
// Compress 压缩数据
func Compress(t Type, data []byte) ([]byte, error)

// Decompress 解压数据
func Decompress(t Type, data []byte) ([]byte, error)
```

**内部实现**:
- GZIP: 使用 `compress/gzip` 标准库
- Snappy: 使用 `snappy.Encode/Decode` (无需 buffer)
- LZ4: 使用 `lz4.Writer/Reader` (需要 buffer)
- ZSTD: 使用 `zstd.Writer/Reader` (需要 buffer)

### 内存管理

不同压缩算法的内存分配策略：

1. **GZIP**: 使用 `bytes.Buffer` 管理内存，写入时动态扩容
2. **Snappy**: 直接操作字节数组，无需额外 buffer
3. **LZ4**: 使用 `bytes.Buffer` + `lz4.Writer/Reader`
4. **ZSTD**: 使用 `bytes.Buffer` + `zstd.Writer/Reader`

**优化建议**:
- 对于高频压缩场景，考虑实现 buffer pool
- 对于大数据块，考虑流式压缩（未来优化）

## Kafka 集成

### Record Batch 压缩

在 Kafka 协议中，压缩发生在 **Record Batch 级别**，而非单个 Record：

```
ProduceRequest
└── TopicData[]
    └── PartitionData[]
        └── Records (bytes)  ← 这里是压缩的 Record Batch
```

**Record Batch 结构** (简化):
```
RecordBatch {
    baseOffset: int64
    length: int32
    attributes: int16      ← 包含压缩类型 (低 3 位)
    lastOffsetDelta: int32
    firstTimestamp: int64
    maxTimestamp: int64
    records: []Record      ← 多个 Record 一起压缩
}
```

**压缩属性编码**:
```go
attributes := int16(compressionType) | (timestampType << 3)
// 低 3 位: 压缩类型 (0-4)
// 第 3 位: 时间戳类型 (0=CreateTime, 1=LogAppendTime)
```

### Handler 集成

当前实现中，Records 已经是 `[]byte` 格式传递，包含完整的 Record Batch（可能已压缩）：

```go
// pkg/kafka/handler/handler.go
func (h *Handler) handleProduce(r io.Reader, header *protocol.RequestHeader) ([]byte, error) {
    // ...
    for _, partData := range topicData.PartitionData {
        // partData.Records 已经是完整的 Record Batch (可能已压缩)
        offset, err := h.backend.Append(
            topicData.TopicName, 
            partData.PartitionIndex, 
            nil, 
            partData.Records, // ← 直接存储原始字节
        )
        // ...
    }
}
```

**设计说明**:
- Kafka 客户端负责压缩 Record Batch
- Takhin 存储原始字节（已压缩或未压缩）
- 解压由 Kafka 客户端负责

**未来增强**:
- 支持服务端压缩配置（broker 端重新压缩）
- 支持压缩类型转换（例如 GZIP → ZSTD）
- 支持压缩统计和监控

## 测试

### 单元测试

每种压缩类型都有独立测试：

```go
func TestGZIPCompression(t *testing.T)
func TestSnappyCompression(t *testing.T)
func TestLZ4Compression(t *testing.T)
func TestZSTDCompression(t *testing.T)
```

### 综合测试

`TestAllCompressionTypes` 测试所有压缩类型：

```go
func TestAllCompressionTypes(t *testing.T) {
    types := []Type{None, GZIP, Snappy, LZ4, ZSTD}
    for _, compType := range types {
        // 测试压缩 → 解压 → 验证
        // 输出压缩率
    }
}
```

### 性能基准测试

`BenchmarkCompression` 测试所有压缩类型的压缩和解压性能：

```bash
go test -bench=. -benchmem ./pkg/compression/
```

**测试结果** (Intel Core i9-12900HK):
```
BenchmarkCompression/Compress_GZIP-20      14568   86495 ns/op   814101 B/op
BenchmarkCompression/Decompress_GZIP-20   126350    9497 ns/op    58640 B/op
BenchmarkCompression/Compress_Snappy-20  1000000    1182 ns/op     5376 B/op
BenchmarkCompression/Decompress_Snappy-20 1000000   1038 ns/op     4864 B/op
BenchmarkCompression/Compress_LZ4-20      642001    1733 ns/op      794 B/op
BenchmarkCompression/Decompress_LZ4-20      3909  291582 ns/op  4232764 B/op
BenchmarkCompression/Compress_ZSTD-20       3710  296812 ns/op  2346826 B/op
BenchmarkCompression/Decompress_ZSTD-20    65401   19875 ns/op    38112 B/op
```

## 监控和指标

### 建议的监控指标

**压缩相关指标**:
- `takhin_compression_bytes_total{type="gzip|snappy|lz4|zstd"}` - 总压缩字节数
- `takhin_compression_ratio{type="gzip|snappy|lz4|zstd"}` - 压缩率
- `takhin_compression_duration_seconds{type="gzip|snappy|lz4|zstd",op="compress|decompress"}` - 压缩/解压耗时
- `takhin_compression_errors_total{type="gzip|snappy|lz4|zstd"}` - 压缩错误数

**使用分布指标**:
- `takhin_compression_usage_total{type="gzip|snappy|lz4|zstd"}` - 各压缩类型使用次数
- `takhin_compression_message_size_bytes{type="gzip|snappy|lz4|zstd"}` - 消息大小分布

### 日志记录

**建议的日志级别**:
- `DEBUG`: 每次压缩/解压操作的详细信息
- `INFO`: 压缩类型切换、配置变更
- `WARN`: 压缩率异常（过低或过高）
- `ERROR`: 压缩/解压失败

**日志示例**:
```go
logger.Debug("compress message", 
    "type", compressionType, 
    "original_size", len(data), 
    "compressed_size", len(compressed),
    "ratio", float64(len(compressed))/float64(len(data)))
```

## 性能优化

### 已实现的优化

1. ✅ **零拷贝设计**: 直接操作字节数组
2. ✅ **最优库选择**: 
   - Snappy: `golang/snappy` (Google 官方)
   - LZ4: `pierrec/lz4` (性能最优)
   - ZSTD: `klauspost/compress` (Pure Go，性能优异)
3. ✅ **Buffer 管理**: 合理使用 `bytes.Buffer`

### 未来优化方向

1. **Buffer Pool**: 
   ```go
   var bufferPool = sync.Pool{
       New: func() interface{} {
           return new(bytes.Buffer)
       },
   }
   ```

2. **流式压缩**: 支持大数据块的流式处理

3. **并行压缩**: 对于大 Batch，支持并行压缩多个 Record

4. **自适应压缩**: 根据数据特征自动选择最优压缩算法

5. **硬件加速**: 利用 CPU 指令集加速（SSE, AVX）

## 故障排查

### 常见问题

**1. 解压失败 "unexpected EOF"**
- **原因**: 压缩数据不完整或损坏
- **解决**: 检查数据传输是否完整，验证数据校验和

**2. 内存占用过高**
- **原因**: LZ4/ZSTD 解压分配大量内存
- **解决**: 考虑实现 buffer pool，限制并发解压数量

**3. 压缩率低于预期**
- **原因**: 数据本身已压缩（如图片、视频）或随机数据
- **解决**: 检查数据特征，考虑不使用压缩

**4. 性能不符合预期**
- **原因**: CPU 资源不足，或数据块太小
- **解决**: 增加 CPU 资源，使用批量压缩，选择更快的算法

### 调试工具

**测试压缩效果**:
```bash
go test -v ./pkg/compression/ -run TestAllCompressionTypes
```

**性能基准测试**:
```bash
go test -bench=. -benchmem -benchtime=10s ./pkg/compression/
```

**CPU Profiling**:
```bash
go test -bench=. -cpuprofile=cpu.out ./pkg/compression/
go tool pprof cpu.out
```

**内存 Profiling**:
```bash
go test -bench=. -memprofile=mem.out ./pkg/compression/
go tool pprof mem.out
```

## 参考资料

### 相关文档
- [Kafka Protocol Guide](https://kafka.apache.org/protocol.html)
- [Kafka Message Format](https://kafka.apache.org/documentation/#messageformat)
- [Compression Benchmark](https://github.com/powturbo/TurboBench)

### 依赖库
- [golang/snappy](https://github.com/golang/snappy) - Snappy 压缩
- [pierrec/lz4](https://github.com/pierrec/lz4) - LZ4 压缩
- [klauspost/compress](https://github.com/klauspost/compress) - ZSTD 和其他压缩算法

### 相关 Issue 和 PR
- 待补充

## 版本历史

### v0.1.0 (2025-12-16)
- ✅ 实现 5 种压缩类型 (None, GZIP, Snappy, LZ4, ZSTD)
- ✅ 完整的单元测试覆盖
- ✅ 性能基准测试
- ✅ 文档完善

### 未来计划
- [ ] Buffer pool 优化
- [ ] 流式压缩支持
- [ ] 服务端压缩配置
- [ ] 压缩统计和监控集成
- [ ] 压缩类型自动选择
