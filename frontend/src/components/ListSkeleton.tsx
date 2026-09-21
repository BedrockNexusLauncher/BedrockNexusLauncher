import React, { useEffect, useState } from "react";

export const useDelayedSkeleton = (delayMs = 200) => {
  const [show, setShow] = useState(false);
  useEffect(() => {
    const id = window.setTimeout(() => setShow(true), delayMs);
    return () => window.clearTimeout(id);
  }, [delayMs]);
  return show;
};

export const ListSkeleton: React.FC<{ rows?: number; className?: string }> = ({
  rows = 6,
  className = "",
}) => {
  const show = useDelayedSkeleton(200);
  if (!show) return null;
  return (
    <div className={`w-full flex flex-col gap-3 p-4 ${className}`} aria-hidden="true">
      {Array.from({ length: rows }).map((_, i) => (
        <div
          key={i}
          className="w-full h-16 rounded-2xl bg-default-100/70 dark:bg-zinc-800/50 animate-pulse"
        />
      ))}
    </div>
  );
};

export default ListSkeleton;
