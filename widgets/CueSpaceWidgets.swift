// Cue Space widgets: your pinned Cues on the desktop, one click to start.
//
// The widget never runs anything itself. It reads a small snapshot the app
// writes (names, icons, item counts: no paths or URLs) and every Cue is a
// cuespace://start/<id> link, so the app's runner stays the only code that
// opens things.

import SwiftUI
import WidgetKit

// MARK: - Data

struct CueItem: Codable, Identifiable, Hashable {
    let id: String
    let name: String
    let icon: String
    let items: Int
    let shortcut: String?
    /// "3 items", in the app's language.
    var detail: String? = nil

    var itemsText: String { detail ?? "\(items) item\(items == 1 ? "" : "s")" }
}

/// The widget's words, written by the app in its current language.
struct Labels: Codable {
    var noCues = "No Cues yet"
    var describeOne = "Describe one in Cue Space."
    var startCue = "Start Cue"
    var cues = "Cues"
}

var labels = Labels()

struct Snapshot: Codable {
    let cues: [CueItem]
    var labels: Labels? = nil
}

/// Reads the snapshot from the real home folder. In the sandbox,
/// FileManager's home is the container, so ask the system for the real one.
func loadCues() -> [CueItem] {
    guard let pw = getpwuid(getuid()), let home = pw.pointee.pw_dir else { return [] }
    let url = URL(fileURLWithPath: String(cString: home))
        .appendingPathComponent("Library/Application Support/CueSpace/widget.json")
    guard let data = try? Data(contentsOf: url),
          let snap = try? JSONDecoder().decode(Snapshot.self, from: data)
    else { return [] }
    labels = snap.labels ?? Labels()
    return snap.cues
}

let sample: [CueItem] = [
    CueItem(id: "1", name: "Atlas", icon: "🛰️", items: 5, shortcut: "⌘1"),
    CueItem(id: "2", name: "Math", icon: "📐", items: 4, shortcut: "⌘2"),
    CueItem(id: "3", name: "Design", icon: "🎨", items: 3, shortcut: "⌘3"),
    CueItem(id: "4", name: "Research", icon: "🔎", items: 6, shortcut: "⌘4"),
]

// MARK: - Timeline

struct CuesEntry: TimelineEntry {
    let date: Date
    let cues: [CueItem]
}

struct Provider: TimelineProvider {
    func placeholder(in context: Context) -> CuesEntry {
        CuesEntry(date: .now, cues: sample)
    }

    func getSnapshot(in context: Context, completion: @escaping (CuesEntry) -> Void) {
        let cues = loadCues()
        completion(CuesEntry(date: .now, cues: context.isPreview && cues.isEmpty ? sample : cues))
    }

    func getTimeline(in context: Context, completion: @escaping (Timeline<CuesEntry>) -> Void) {
        // The app asks for a reload whenever Cues change; this is a fallback.
        let entry = CuesEntry(date: .now, cues: loadCues())
        completion(Timeline(entries: [entry], policy: .after(.now.addingTimeInterval(15 * 60))))
    }
}

// MARK: - Style

extension Color {
    static let cueRed = Color(red: 217 / 255, green: 48 / 255, blue: 37 / 255)
}

func startURL(_ cue: CueItem) -> URL {
    URL(string: "cuespace://start/\(cue.id)")!
}

let openURL = URL(string: "cuespace://open")!

/// The ring-and-dot mark.
struct CueMark: View {
    var size: CGFloat = 14
    var color: Color = .cueRed

    var body: some View {
        Canvas { ctx, s in
            let k = s.width / 100
            var ring = Path()
            ring.addArc(center: CGPoint(x: 49.5 * k, y: 50 * k), radius: 30 * k,
                        startAngle: .degrees(-40), endAngle: .degrees(40), clockwise: true)
            ctx.stroke(ring, with: .color(color), style: StrokeStyle(lineWidth: 10 * k, lineCap: .round))
            ctx.fill(Path(ellipseIn: CGRect(x: 71.3 * k, y: 43 * k, width: 14 * k, height: 14 * k)), with: .color(color))
        }
        .frame(width: size, height: size)
    }
}

// MARK: - Views

struct EmptyView: View {
    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            CueMark(size: 22)
            Spacer()
            Text(labels.noCues).font(.headline)
            Text(labels.describeOne).font(.caption).foregroundStyle(.secondary)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .leading)
        .widgetURL(openURL)
    }
}

/// Small: the first Cue, big and tappable.
struct SmallView: View {
    let cue: CueItem

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            HStack {
                Text(cue.icon).font(.system(size: 30))
                Spacer()
                CueMark(size: 16)
            }
            Spacer()
            Text(cue.name).font(.system(size: 17, weight: .semibold)).lineLimit(1)
            Text(cue.itemsText).font(.caption).foregroundStyle(.secondary)
            HStack(spacing: 4) {
                Text(labels.startCue).font(.system(size: 12, weight: .semibold))
                Image(systemName: "arrow.right").font(.system(size: 10, weight: .bold))
            }
            .foregroundStyle(.white)
            .frame(maxWidth: .infinity)
            .padding(.vertical, 6)
            .background(Capsule().fill(Color.cueRed))
            .padding(.top, 8)
        }
        .widgetURL(startURL(cue))
    }
}

/// One tappable row for medium and large widgets.
struct CueRow: View {
    let cue: CueItem

    var body: some View {
        Link(destination: startURL(cue)) {
            HStack(spacing: 10) {
                Text(cue.icon)
                    .font(.system(size: 18))
                    .frame(width: 32, height: 32)
                    .background(RoundedRectangle(cornerRadius: 9, style: .continuous).fill(.quaternary))
                VStack(alignment: .leading, spacing: 1) {
                    Text(cue.name).font(.system(size: 13, weight: .semibold)).lineLimit(1)
                    Text(cue.itemsText).font(.system(size: 11)).foregroundStyle(.secondary)
                }
                Spacer(minLength: 4)
                Image(systemName: "play.fill")
                    .font(.system(size: 9, weight: .bold))
                    .foregroundStyle(.white)
                    .frame(width: 22, height: 22)
                    .background(Circle().fill(Color.cueRed))
            }
        }
    }
}

struct ListView: View {
    let cues: [CueItem]
    let limit: Int
    let columns: Int

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack(spacing: 6) {
                CueMark(size: 14)
                Text(labels.cues).font(.system(size: 13, weight: .semibold))
                Spacer()
            }
            LazyVGrid(columns: Array(repeating: GridItem(.flexible(), spacing: 12), count: columns), alignment: .leading, spacing: 8) {
                ForEach(cues.prefix(limit)) { CueRow(cue: $0) }
            }
            Spacer(minLength: 0)
        }
    }
}

struct CuesWidgetView: View {
    @Environment(\.widgetFamily) var family
    let entry: CuesEntry

    var body: some View {
        Group {
            if entry.cues.isEmpty {
                EmptyView()
            } else {
                switch family {
                case .systemSmall: SmallView(cue: entry.cues[0])
                case .systemMedium: ListView(cues: entry.cues, limit: 4, columns: 2)
                case .systemLarge: ListView(cues: entry.cues, limit: 8, columns: 1)
                default: ListView(cues: entry.cues, limit: 12, columns: 2)
                }
            }
        }
        .containerBackground(for: .widget) { Color(nsColor: .windowBackgroundColor) }
    }
}

struct CuesWidget: Widget {
    var body: some WidgetConfiguration {
        StaticConfiguration(kind: "space.cue.app.cues", provider: Provider()) { entry in
            CuesWidgetView(entry: entry)
        }
        .configurationDisplayName("Cues")
        .description("Your pinned Cues. Click one to start it.")
        .supportedFamilies([.systemSmall, .systemMedium, .systemLarge, .systemExtraLarge])
    }
}

@main
struct CueSpaceWidgets: WidgetBundle {
    var body: some Widget {
        CuesWidget()
    }
}
