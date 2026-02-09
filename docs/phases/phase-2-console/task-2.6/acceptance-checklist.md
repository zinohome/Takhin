# Task 2.6: Message Browser - Acceptance Testing Checklist

## Test Environment Setup
- [ ] Backend Takhin server running
- [ ] Backend Console API running
- [ ] Frontend dev server running or built assets served
- [ ] At least one topic created with messages
- [ ] Multiple partitions (optional but recommended)

## Core Functionality Tests

### 1. Partition Message List ✓
**Acceptance Criteria**: Display messages from selected partition in table format

- [ ] Navigate to Topics page
- [ ] Click "Messages" button on a topic
- [ ] Verify page loads with topic name in header
- [ ] Verify topic info card shows partition count
- [ ] Click "Filter" button
- [ ] Select a partition from dropdown
- [ ] Click "Apply & Load"
- [ ] Verify table displays messages with columns:
  - [ ] Partition (blue tag)
  - [ ] Offset (sortable number)
  - [ ] Timestamp (formatted date)
  - [ ] Key (code style, ellipsis if long)
  - [ ] Value (code style, ellipsis if long)
  - [ ] Actions (View button)
- [ ] Verify pagination controls appear
- [ ] Change page size (20/50/100/200)
- [ ] Navigate through pages if multiple pages exist

**Expected Results**:
- Messages display in table format
- All columns visible and properly formatted
- Pagination works correctly
- No console errors

---

### 2. Offset Range Query ✓
**Acceptance Criteria**: Query messages by offset range (start and/or end)

- [ ] Open filter modal
- [ ] Enter start offset (e.g., 100)
- [ ] Leave end offset empty
- [ ] Click "Apply & Load"
- [ ] Verify messages shown are >= start offset
- [ ] Open filter again
- [ ] Set both start (e.g., 100) and end offset (e.g., 200)
- [ ] Click "Apply & Load"
- [ ] Verify messages shown are between start and end offsets
- [ ] Test with start offset = 0
- [ ] Test with very large offset (beyond HWM)

**Expected Results**:
- Messages filtered by offset range correctly
- Start-only filter works
- Start+end filter works
- Out of range returns empty or partial results
- No errors with edge cases

---

### 3. Time Range Query ✓
**Acceptance Criteria**: Query messages by timestamp range

- [ ] Open filter modal
- [ ] Click on "Time Range" date picker
- [ ] Select start date/time
- [ ] Select end date/time
- [ ] Click "Apply & Load"
- [ ] Verify only messages within time range displayed
- [ ] Check timestamp column values match filter
- [ ] Test with narrow time range (minutes)
- [ ] Test with wide time range (days)
- [ ] Test with time range outside message timestamps

**Expected Results**:
- Messages filtered by timestamp correctly
- Date picker works properly
- Time selection functional
- Timestamps in table match filter criteria
- Empty result if no messages in range

---

### 4. Key/Value Search ✓
**Acceptance Criteria**: Search messages by key or value substring

#### Key Search
- [ ] Load messages for a partition
- [ ] Open filter modal
- [ ] Enter substring in "Search by Key" field
- [ ] Click "Apply & Load"
- [ ] Verify only messages with matching keys displayed
- [ ] Test with case variations (uppercase/lowercase)
- [ ] Test with partial matches
- [ ] Test with non-existent key

#### Value Search
- [ ] Open filter modal
- [ ] Clear key search
- [ ] Enter substring in "Search by Value" field
- [ ] Click "Apply & Load"
- [ ] Verify only messages with matching values displayed
- [ ] Test with JSON field search (e.g., "email")
- [ ] Test with special characters
- [ ] Test with non-existent value

#### Combined Search
- [ ] Set both key and value searches
- [ ] Verify messages match both criteria (AND logic)

**Expected Results**:
- Key search filters correctly
- Value search filters correctly
- Case-insensitive matching works
- Combined filters work with AND logic
- Empty search returns all messages

---

### 5. JSON Format Display ✓
**Acceptance Criteria**: Automatically detect and pretty-print JSON values

- [ ] Load messages with JSON values
- [ ] Verify green "JSON" badge appears on JSON messages
- [ ] Verify non-JSON messages don't have badge
- [ ] Click "View" on a JSON message
- [ ] Verify drawer shows pretty-printed JSON:
  - [ ] Proper indentation (2 spaces)
  - [ ] Formatted structure
  - [ ] Code block styling
- [ ] Test with nested JSON objects
- [ ] Test with JSON arrays
- [ ] Test with mixed content (some JSON, some plain text)
- [ ] Test with invalid JSON (should show as plain text)

**Expected Results**:
- JSON badge appears correctly
- JSON formatted with indentation in drawer
- Plain text displayed normally
- No errors with malformed JSON

---

### 6. Message Details View ✓
**Acceptance Criteria**: View comprehensive message details in side drawer

- [ ] Click "View" button on any message
- [ ] Verify drawer opens from right side
- [ ] Verify drawer shows all fields:
  - [ ] Partition (with blue tag)
  - [ ] Offset (with copy icon)
  - [ ] Timestamp (human-readable + Unix)
  - [ ] Key (code style or "(empty)")
  - [ ] Value (formatted or plain text)
- [ ] Click copy icon on offset
- [ ] Verify offset copied to clipboard
- [ ] Test with message containing empty key
- [ ] Test with message containing very long value
- [ ] Test with JSON value (verify formatting)
- [ ] Close drawer with X button
- [ ] Close drawer by clicking outside
- [ ] Press Escape key to close drawer

**Expected Results**:
- Drawer opens smoothly
- All message fields visible
- Copy functionality works
- JSON formatted correctly
- Drawer closes properly
- No layout issues with long content

---

### 7. Export Functionality ✓
**Acceptance Criteria**: Export messages to JSON file

#### Bulk Export (Table)
- [ ] Load messages (at least 10)
- [ ] Click "Export" button in toolbar
- [ ] Verify download starts
- [ ] Check downloaded file:
  - [ ] Filename format: `{topic}_messages_{timestamp}.json`
  - [ ] Valid JSON format
  - [ ] Contains all filtered messages
  - [ ] Pretty-printed (indented)
  - [ ] All message fields present
- [ ] Test export with 1 message
- [ ] Test export with 100+ messages
- [ ] Test export with no messages (should show warning)
- [ ] Test export with filtered results

#### Single Message Export (Drawer)
- [ ] Open message detail drawer
- [ ] Click "Export Message" button
- [ ] Verify download starts
- [ ] Check downloaded file:
  - [ ] Filename format: `message_{partition}_{offset}.json`
  - [ ] Valid JSON format
  - [ ] Contains single message object
  - [ ] Pretty-printed

**Expected Results**:
- Bulk export downloads all filtered messages
- Single export downloads one message
- Filenames are descriptive and unique
- JSON is valid and readable
- No export errors
- Success notification appears

---

## Integration Tests

### Navigation Flow
- [ ] Start at Topics page
- [ ] Click "Messages" on topic A
- [ ] Verify message browser loads for topic A
- [ ] Click "Back" button
- [ ] Verify return to Topics page
- [ ] Click "Messages" on topic B
- [ ] Verify message browser loads for topic B
- [ ] Browser back button works correctly
- [ ] Direct URL navigation works: `/topics/{topic}/messages`

### Filter Combinations
- [ ] Partition + Offset range
- [ ] Partition + Time range
- [ ] Partition + Key search
- [ ] Partition + Value search
- [ ] All filters combined
- [ ] Reset filters button clears all
- [ ] Filter state persists during session
- [ ] Filter resets on topic change

### Error Handling
- [ ] Navigate to non-existent topic
- [ ] Query partition that doesn't exist
- [ ] Query offset beyond HWM
- [ ] Network timeout during load
- [ ] Backend API unavailable
- [ ] Invalid filter values
- [ ] Empty topic (no messages)

---

## UI/UX Tests

### Responsiveness
- [ ] Test at 1920x1080 resolution
- [ ] Test at 1280x720 resolution
- [ ] Test with narrow browser window
- [ ] Verify table scrolls horizontally if needed
- [ ] Drawer doesn't break layout
- [ ] Modal is centered and responsive

### Loading States
- [ ] Loading spinner appears during fetch
- [ ] Table shows loading overlay
- [ ] Filter modal disabled during load
- [ ] Buttons disabled appropriately
- [ ] Loading completes or times out

### Empty States
- [ ] Empty topic shows helpful message
- [ ] No search results shows message
- [ ] Suggests adjusting filters
- [ ] Provides action to reset

### Notifications
- [ ] Success: Topic loaded
- [ ] Success: Messages loaded
- [ ] Success: Export complete
- [ ] Error: Failed to load topic
- [ ] Error: Failed to load messages
- [ ] Warning: No messages to export
- [ ] Notifications auto-dismiss
- [ ] Multiple notifications stack properly

---

## Performance Tests

### Load Time
- [ ] Small dataset (< 100 messages): < 1 second
- [ ] Medium dataset (100-500 messages): < 2 seconds
- [ ] Large dataset (500+ messages): < 5 seconds
- [ ] Filter application: < 500ms
- [ ] Export: < 2 seconds for 1000 messages

### Memory
- [ ] No memory leaks after 10 navigations
- [ ] Browser memory stable after multiple exports
- [ ] No console warnings about memory

### Rendering
- [ ] Table renders smoothly
- [ ] Pagination doesn't cause flicker
- [ ] Drawer animation smooth
- [ ] Modal transitions smooth

---

## Cross-Browser Tests

### Chrome
- [ ] All features work
- [ ] No console errors
- [ ] Export downloads correctly

### Firefox
- [ ] All features work
- [ ] No console errors
- [ ] Export downloads correctly

### Safari
- [ ] All features work
- [ ] No console errors
- [ ] Export downloads correctly

### Edge
- [ ] All features work
- [ ] No console errors
- [ ] Export downloads correctly

---

## Accessibility Tests

### Keyboard Navigation
- [ ] Tab through all interactive elements
- [ ] Enter/Space activate buttons
- [ ] Escape closes modals/drawers
- [ ] Arrow keys navigate table
- [ ] Focus visible on all elements

### Screen Reader
- [ ] Labels read correctly
- [ ] Buttons have descriptive text
- [ ] Error messages announced
- [ ] Table structure announced

### Visual
- [ ] Text readable at 100% zoom
- [ ] Text readable at 200% zoom
- [ ] Sufficient color contrast
- [ ] No color-only indicators

---

## Edge Cases

### Data Edge Cases
- [ ] Message with null/empty key
- [ ] Message with null/empty value
- [ ] Message with very long key (> 1000 chars)
- [ ] Message with very long value (> 100KB)
- [ ] Message with special characters
- [ ] Message with Unicode/emoji
- [ ] Message with binary data (escaped)
- [ ] Offset at max int64 value
- [ ] Timestamp at Unix epoch (0)
- [ ] Timestamp in future

### Filter Edge Cases
- [ ] Start offset = End offset
- [ ] End offset < Start offset (should handle gracefully)
- [ ] Time range with same start/end
- [ ] Search with regex special characters
- [ ] Search with very long string
- [ ] Empty search string (should show all)

### System Edge Cases
- [ ] Topic with 0 partitions (shouldn't exist)
- [ ] Topic with 100+ partitions
- [ ] Partition with 0 messages
- [ ] Partition with millions of messages
- [ ] Very fast message production during viewing
- [ ] Topic deleted while viewing

---

## Regression Tests

### Previous Features
- [ ] Topics page still works
- [ ] Topic creation still works
- [ ] Topic deletion still works
- [ ] Dashboard not affected
- [ ] Consumers page not affected
- [ ] Brokers page not affected
- [ ] API client still functional

### Dependencies
- [ ] dayjs formatting works
- [ ] Ant Design components render
- [ ] React Router navigation works
- [ ] TypeScript types enforced

---

## Sign-Off Checklist

### Development
- [x] Code complete
- [x] TypeScript compilation successful
- [x] ESLint passes with no warnings
- [x] Production build successful
- [x] No console errors in dev mode

### Testing
- [ ] All core functionality tests passed
- [ ] All integration tests passed
- [ ] All UI/UX tests passed
- [ ] Performance acceptable
- [ ] Cross-browser testing complete
- [ ] Accessibility requirements met
- [ ] Edge cases handled

### Documentation
- [x] Completion summary created
- [x] Quick reference guide created
- [x] Code commented appropriately
- [x] Component API documented
- [x] Known limitations documented

### Deployment
- [ ] Build artifacts generated
- [ ] No security vulnerabilities
- [ ] Bundle size acceptable
- [ ] Dependencies up to date
- [ ] Ready for production

---

## Test Results Summary

**Date**: __________  
**Tester**: __________  
**Environment**: __________

### Results
- Total Tests: _____ / _____
- Passed: _____
- Failed: _____
- Blocked: _____
- Skipped: _____

### Critical Issues Found
1. ___________________________________
2. ___________________________________
3. ___________________________________

### Non-Critical Issues Found
1. ___________________________________
2. ___________________________________
3. ___________________________________

### Recommendation
- [ ] **APPROVED** - Ready for production
- [ ] **APPROVED WITH ISSUES** - Deploy with known issues
- [ ] **REJECTED** - Critical issues must be fixed

**Approver Signature**: __________  
**Date**: __________

---

**Version**: 1.0  
**Task**: 2.6 Message Browser  
**Status**: Pending QA  
