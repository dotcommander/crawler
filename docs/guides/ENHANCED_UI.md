# Enhanced Bubbletea UI for Crawler (Default)

## Overview

The enhanced UI is now the **default interface** for the crawler, providing a beautiful, responsive terminal experience with proper Bubbletea architecture that fixes all the garbling issues from the previous SimpleUI implementation.

## Features

- **Proper Bubbletea Architecture**: All UI updates go through the message system
- **Responsive Design**: Handles terminal resizing gracefully
- **Scrollable Content**: Uses viewport for logs and stats that exceed screen height
- **Beautiful Animations**: Smooth progress bars, animated spinners, and visual effects
- **Keyboard Navigation**: Full keyboard support with help screen
- **Terminal State Management**: Proper cleanup and restoration on exit
- **High Performance**: Throttled rendering to prevent flicker

## Usage

### Default Behavior

```bash
# The enhanced UI is now the default!
./crawler https://example.com

# To use verbose mode (simple logging, no UI)
./crawler -v https://example.com

# To use the legacy SimpleUI (if needed)
export CRAWLER_LEGACY_UI=1
./crawler https://example.com
```

### Keyboard Controls

- `q` or `Ctrl+C`: Quit the application
- `?` or `h`: Toggle help screen
- `Tab`: Switch between panes
- `↑/↓` or `j/k`: Scroll current pane
- `PgUp/PgDn`: Page up/down

### UI Sections

1. **Header**: Shows title, current URL, and elapsed time
2. **Progress Bar**: Real-time crawling progress with percentage
3. **Statistics**: Live metrics including pages visited, queue size, etc.
4. **Worker Status**: Shows activity of concurrent workers
5. **Activity Log**: Recent operations and errors (scrollable)
6. **Footer**: Help text and current status

## Architecture Improvements

### 1. Message-Based Updates
All UI updates now go through Bubbletea's message system:
```go
// Instead of direct terminal manipulation
fmt.Print("\033[9A") // BAD

// Now using proper messages
program.Send(ui.StatsMsg{...}) // GOOD
```

### 2. Viewport for Scrolling
Content that exceeds screen space is handled by viewports:
```go
viewport := viewport.New(width, height)
viewport.SetContent(content)
```

### 3. Terminal State Management
Proper cleanup ensures terminal is restored:
```go
defer func() {
    term.Restore(fd, oldState)
    fmt.Print("\033[?25h")  // Show cursor
    fmt.Print("\033[0m")    // Reset colors
}()
```

### 4. Responsive Design
Handles window resize events properly:
```go
case tea.WindowSizeMsg:
    m.width = msg.Width
    m.height = msg.Height
    m.updateViewportContent()
```

## Visual Design

- **Color Scheme**: Cyberpunk/Matrix theme
  - Primary: `#00ff41` (Matrix green)
  - Secondary: `#39ff14` (Neon green)
  - Accent: `#ff006e` (Hot pink)
  - Warning: `#ffb700` (Amber)
  - Error: `#ff0040` (Red)

- **Animations**: 
  - Smooth progress bars
  - Rotating spinners for active workers
  - Wave effects for URLs being processed
  - Sparkle effects in headers

## Comparison with SimpleUI

| Feature | SimpleUI | Enhanced UI |
|---------|----------|-------------|
| Architecture | Direct ANSI codes | Bubbletea messages |
| Terminal Safety | Hardcoded assumptions | Dynamic adaptation |
| Scrolling | None | Full viewport support |
| Resize Handling | Breaks layout | Responsive reflow |
| Cleanup | None | Full state restoration |
| Performance | Can flicker | Throttled rendering |

## Future Enhancements

1. **Mouse Support**: Click on URLs to copy, scroll with mouse wheel
2. **Themes**: Multiple color schemes to choose from
3. **Export**: Save crawl results in various formats
4. **Filtering**: Real-time log filtering by level or content
5. **Charts**: Visual representation of crawl progress over time

## Troubleshooting

If you experience any issues:

1. **Garbled Output**: Make sure `CRAWLER_ENHANCED_UI=1` is set
2. **Colors Not Working**: Check if your terminal supports 256 colors
3. **Resize Issues**: Try a different terminal emulator
4. **Performance**: Adjust render throttling in the code if needed

## Development

To modify the enhanced UI:

1. Edit `ui/enhanced_model.go` for the main model
2. Update `ui/styles.go` for visual styling
3. Modify `internal/crawlers/crawler_enhanced_ui.go` for integration
4. Test with various terminal sizes and emulators