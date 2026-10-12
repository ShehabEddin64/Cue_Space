// cuespace-widget-reload: asks WidgetKit to refresh Cue Space's widgets.
// The Go app runs it (from Contents/MacOS, so it belongs to the app bundle)
// after writing a new widget snapshot. WidgetCenter has no C or Objective-C
// API, hence this tiny Swift helper.
import WidgetKit

WidgetCenter.shared.reloadAllTimelines()
