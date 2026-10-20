const MINUTE_MS = 60_000;

function timeFormat(timeZone?: string): Intl.DateTimeFormat {
  return new Intl.DateTimeFormat('en-GB', { hour: '2-digit', minute: '2-digit', timeZone });
}

/** The 24-hour time in the local time zone, such as "01:12". */
export function timeLabel(date: Date): string {
  return timeFormat().format(date);
}

/** The top bar's clock, such as "01:12 · Tokyo": the 24-hour time and the time zone's city, local by default. */
export function clockLabel(now: Date, timeZone?: string): string {
  const format = timeFormat(timeZone);
  return `${format.format(now)} · ${cityOf(format.resolvedOptions().timeZone)}`;
}

// The last part of an IANA time zone, so "America/Argentina/Buenos_Aires" gives "Buenos Aires".
function cityOf(timeZone: string): string {
  return timeZone.slice(timeZone.lastIndexOf('/') + 1).replaceAll('_', ' ');
}

/** Calls onTick at the start of every minute until the returned function is called. */
export function startClock(onTick: (now: Date) => void): () => void {
  let timer: ReturnType<typeof setTimeout>;
  const schedule = () => {
    timer = setTimeout(
      () => {
        onTick(new Date());
        schedule();
      },
      MINUTE_MS - (Date.now() % MINUTE_MS)
    );
  };
  schedule();
  return () => clearTimeout(timer);
}
