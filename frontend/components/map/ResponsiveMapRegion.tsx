import type { ReactNode } from "react";

export function ResponsiveMapRegion({
  mobile,
  listRail,
  renderMap,
}: {
  mobile: boolean;
  listRail: ReactNode;
  renderMap: (mobile: boolean) => ReactNode;
}) {
  if (mobile) {
    return (
      <div className="flex h-[calc(100vh-210px)] min-h-[360px] flex-col border-b border-ink sm:hidden">
        {renderMap(true)}
      </div>
    );
  }

  return (
    <div className="hidden h-[calc(100vh-160px)] min-h-[420px] grid-cols-[230px_1fr] border-b border-ink sm:grid">
      <div className="flex flex-col overflow-y-auto border-r border-on-surface">{listRail}</div>
      {renderMap(false)}
    </div>
  );
}
