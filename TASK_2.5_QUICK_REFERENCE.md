# Task 2.5: Consumer Group Monitoring - Quick Reference

## 🚀 Quick Start

### View Consumer Groups
```
Navigate to: /consumers
Auto-refresh: Toggle ON/OFF (5s interval)
```

### View Group Details
```
Click: Group ID in list
Route: /consumers/:groupId
Back: Click "Back" button or browser back
```

---

## 📁 File Structure

```
frontend/src/
├── pages/
│   └── Consumers.tsx              # Main list + routing logic
├── components/
│   ├── ConsumerGroupDetail.tsx    # Detail view component
│   └── LagChart.tsx                # Lag visualization
└── api/
    ├── takhinApi.ts                # API client (existing)
    └── types.ts                    # Type definitions (existing)
```

---

## 🔑 Key Features

### List View
| Feature | Implementation |
|---------|---------------|
| Group listing | API: `GET /api/consumer-groups` |
| Lag data | API: `GET /api/monitoring/metrics` |
| Auto-refresh | 5-second interval (toggle) |
| Sorting | Click column headers |
| Navigation | Click group ID |

### Detail View
| Component | Description |
|-----------|-------------|
| Statistics | State, Members, Topics, Total Lag |
| Group Info | Protocol details, state |
| Lag Chart | Visual representation per topic/partition |
| Members | Client IDs, hosts, partitions |
| Offsets | Current offset, lag, log end offset |

---

## 🎨 Color Coding

### Lag Indicators
- 🟢 Green: `lag < 100` (Healthy)
- 🟠 Orange: `100 ≤ lag < 1000` (Warning)
- 🔴 Red: `lag ≥ 1000` (Critical)

### State Tags
- 🟢 Green: `Stable`
- 🟠 Orange: `Rebalancing`
- 🔴 Red: `Dead`
- ⚫ Gray: `Empty`

---

## 🔧 API Integration

### Endpoints Used
```typescript
// List groups
takhinApi.listConsumerGroups()
→ ConsumerGroupSummary[]

// Get group details
takhinApi.getConsumerGroup(groupId)
→ ConsumerGroupDetail

// Get lag metrics
takhinApi.getMonitoringMetrics()
→ MonitoringMetrics (includes consumerLags)
```

### Data Flow
```
List View:
  ├─ Fetch groups + metrics in parallel
  ├─ Map lag data to groups
  └─ Render table

Detail View:
  ├─ Fetch group detail + metrics in parallel
  ├─ Correlate offsets with lag data
  ├─ Calculate per-partition lag
  └─ Render charts + tables
```

---

## 📊 Component Props

### ConsumerGroupDetail
```typescript
interface ConsumerGroupDetailProps {
  groupId: string          // Group to display
  onRefresh?: () => void   // Optional callback (unused currently)
}
```

### LagChart
```typescript
interface LagChartProps {
  lag: ConsumerGroupLag    // Lag data structure
}
```

---

## 🐛 Troubleshooting

### Issue: No groups showing
- **Check**: Backend server running?
- **Check**: API key configured (if auth enabled)?
- **Check**: Browser console for errors

### Issue: Lag shows "-"
- **Cause**: No offset commits yet
- **Solution**: Wait for consumer to commit offsets

### Issue: Partitions show "None"
- **Cause**: Backend TODO (assignment parsing)
- **Status**: Known limitation, backend fix pending

### Issue: Auto-refresh not working
- **Check**: Toggle button is "ON"
- **Check**: No console errors blocking updates
- **Fix**: Click manual refresh button

---

## 🧪 Testing Checklist

### Manual Testing
- [ ] List page loads without errors
- [ ] Groups display with correct data
- [ ] Lag values show with colors
- [ ] Click group navigates to detail
- [ ] Detail page shows all sections
- [ ] Lag chart renders correctly
- [ ] Auto-refresh updates data
- [ ] Back button returns to list
- [ ] Pagination works for large datasets
- [ ] Empty states display properly

### Browser Console
```bash
# Should see no errors
# Expected API calls every 5 seconds when auto-refresh ON
```

---

## 🚀 Build & Deploy

### Development
```bash
cd frontend
npm install
npm run dev
# Visit: http://localhost:5173/consumers
```

### Production Build
```bash
npm run build
# Output: frontend/dist/
```

### Lint Check
```bash
npm run lint
# Should pass with 0 errors
```

---

## 📝 Code Snippets

### Add Custom Lag Threshold
```typescript
// In Consumers.tsx or ConsumerGroupDetail.tsx
const getLagColor = (lag: number) => {
  if (lag > 5000) return 'red'      // Custom threshold
  if (lag > 500) return 'orange'
  return 'green'
}
```

### Change Auto-Refresh Interval
```typescript
// In Consumers.tsx, line ~60
const interval = setInterval(fetchGroups, 10000) // 10 seconds
```

### Add Filtering
```typescript
// In Consumers.tsx, after tableData definition
const filteredData = tableData.filter(g => 
  g.state === 'Stable' // or any condition
)
```

---

## 🔮 Upcoming Features (Phase 2)

### Reset Offset
**Status**: Frontend ready, backend pending

**Implementation**:
```typescript
// Backend endpoint needed
POST /api/consumer-groups/:groupId/reset-offset
Body: {
  topic: string
  partition: number
  offset: number | 'earliest' | 'latest'
}
```

**Frontend Addition**:
- Add reset button in offset table
- Modal for offset selection
- Confirmation dialog
- Success/error toast

---

## 📚 Related Files

### Backend
- `backend/pkg/console/server.go` - API handlers
- `backend/pkg/console/types.go` - Data structures
- `backend/pkg/console/monitoring.go` - Lag calculation
- `backend/pkg/coordinator/*` - Consumer group logic

### Frontend
- `frontend/src/App.tsx` - Routing config
- `frontend/src/api/takhinApi.ts` - API client
- `frontend/src/api/types.ts` - TypeScript types

---

## 💡 Pro Tips

1. **Monitor Lag Trends**: Keep auto-refresh ON to spot growing lag
2. **Check Members**: Empty members table = no active consumers
3. **Partition Distribution**: Even lag distribution = good load balancing
4. **State Changes**: Rebalancing state is normal during consumer joins/leaves
5. **Performance**: Disable auto-refresh when not actively monitoring

---

## 🆘 Support

### Common Questions

**Q: Why is lag calculation different from Kafka?**
A: Lag = LogEndOffset - CurrentOffset (standard Kafka formula)

**Q: Can I export lag data?**
A: Not yet - planned for Phase 2

**Q: How to reset a stuck consumer?**
A: Reset offset feature coming in Phase 2

**Q: What if group state is "Dead"?**
A: No active members, group may need cleanup

---

## ✅ Feature Checklist

- [x] Consumer group list
- [x] Group state indicators
- [x] Member count display
- [x] Topic subscription list
- [x] Total lag calculation
- [x] Detail page navigation
- [x] Group information display
- [x] Member details table
- [x] Offset commit table
- [x] Lag visualization chart
- [x] Real-time updates
- [x] Auto-refresh toggle
- [ ] Reset offset (pending backend)
- [ ] Historical lag tracking (future)
- [ ] Alert configuration (future)

---

**Status**: ✅ Production Ready
**Version**: 1.0
**Last Updated**: 2026-01-02
