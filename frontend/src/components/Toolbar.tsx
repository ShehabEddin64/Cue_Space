import { PanelLeft } from "lucide-react";
import { createContext, useContext, type ReactNode } from "react";
import { t } from "../i18n";
import { IconButton } from "./ui";

export const LayoutContext = createContext<{ sidebarOpen: boolean; toggleSidebar: () => void }>({
  sidebarOpen: true,
  toggleSidebar: () => {},
});

/**
 * The window's toolbar row for the content pane. It drags the window, leaves
 * room for the traffic lights when the sidebar is hidden, and keeps the
 * Show Sidebar control where people expect it.
 */
export function Toolbar({ leading, title, trailing }: { leading?: ReactNode; title?: ReactNode; trailing?: ReactNode }) {
  const { sidebarOpen, toggleSidebar } = useContext(LayoutContext);
  return (
    <div className={`drag flex h-[52px] shrink-0 items-center gap-2 pr-4 ${sidebarOpen ? "pl-4" : "pl-[84px]"}`}>
      {!sidebarOpen && (
        <IconButton label={t("Show Sidebar (⌃⌘S)")} onClick={toggleSidebar}>
          <PanelLeft className="size-4" strokeWidth={1.75} />
        </IconButton>
      )}
      {leading}
      {title && <div className="min-w-0 truncate text-[13px] font-semibold">{title}</div>}
      <div className="flex-1" />
      {trailing}
    </div>
  );
}
