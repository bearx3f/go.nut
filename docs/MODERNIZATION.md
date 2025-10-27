# go.nut - Modernization Summary

## Changes Made (October 2025)

This codebase has been updated to improve compatibility with modern NUT servers and Go best practices.

### 1. ✅ Go Modules Support (go.mod)
- Added proper Go module file for dependency management
- Targets Go 1.21+
- Module name: `github.com/bearx3f/go.nut`

### 2. ✅ TLS/SSL Support via STARTTLS
**New Method**: `Client.StartTLS()`
- Implements STARTTLS for secure connections
- Compatible with NUT 2.7.0+
- Configurable TLS settings via `Client.TLSConfig`
- Example usage:
  ```go
  client.StartTLS()
  ```

**New Client Fields**:
- `UseTLS` - Boolean flag indicating TLS mode
- `TLSConfig` - Optional TLS configuration
- `ConnectTimeout` - Configurable connection timeout (default: 5s)
- `ReadTimeout` - Configurable read timeout (default: 2s)

### 3. ✅ Improved Error Handling
- Enhanced error wrapping with context (`%w`)
- Better validation of server responses
- Safer error code extraction (checks slice bounds)
- Added STARTTLS to recognized commands

### 4. ✅ Better Timeout Management
- Default connection timeout: 5 seconds
- Default read timeout: 2 seconds
- User-customizable per client instance
- ReadResponse now respects configured timeouts

### 5. ✅ Code Quality Improvements
- Fixed boolean comparison lint issue (`!multiLineResponse` instead of `== false`)
- Updated example test to use correct Go syntax
- Fixed import references for the new repository location

### 6. ✅ Updated Documentation
- Comprehensive README with examples
- TLS/SSL usage guide
- Timeout configuration guide
- Compatibility information
- Updated to reference bearx3f/go.nut

## Backward Compatibility
✅ All existing API methods remain unchanged
✅ Existing code using plain TCP will continue to work
✅ TLS features are opt-in

## Testing Recommendations
1. Test with latest NUT server (2.7.0+)
2. Verify STARTTLS handshake on supported servers
3. Test authentication with custom timeouts
4. Validate error handling with edge cases

## Future Improvements (Optional)
- Connection pooling for multiple UPS devices
- Async event listening for UPS state changes
- Structured logging instead of simple error messages
- Unit tests with mocks
- CI/CD pipeline with Go versions 1.21+

## Breaking Changes
None - This is a fully backward-compatible update.

---
Updated: October 27, 2025
