import { loadSnapshot, type Polled, type Polls } from '../shell/snapshot';

/** The trophies or achievements I earned in a game, out of its total. */
interface Trophies {
  earned: number;
  total: number;
  /** How far I got through them, as Exophase reckons it. */
  percent: number;
}

/** A game I played lately, as the games service lists it (decisions 301 and 303). */
export interface Game {
  /** Its name on Exophase. */
  title: string;
  /** The platforms that Exophase lists for it, such as PS5 or Switch. */
  platforms: string[];
  /** When I last played it, in UTC. */
  lastPlayed: string;
  /** How long I've played it; none when Exophase doesn't know. */
  playtimeMinutes?: number;
  /** None for a game without trophies, such as Nintendo's. */
  trophies?: Trophies;
  /** The address of its cover on IGDB, at most 264 by 374 px; none when IGDB has none or isn't set up (decision 306). */
  cover?: string;
}

/** The body of GET /api/games/recent: what the games service last found on Exophase (decision 303). */
export interface Snapshot extends Polls {
  /** The games I played in the last 14 days, the last played first; empty, not missing, before a fetch succeeds. */
  games: Game[];
}

/** The Games page's list: loading, off when Exophase isn't set up, failed to load, or loaded. */
export type Games = Polled<Snapshot>;

/** Loads the games I played lately; never fails, since a failed load is one of the outcomes. */
export function loadGames(): Promise<Games> {
  return loadSnapshot('/api/games/recent');
}

/** The snapshot's games, which only a fetch that succeeded can have found; none before one has. */
export function foundGames(snapshot: Snapshot): Game[] | undefined {
  return snapshot.searched === undefined ? undefined : snapshot.games;
}

/** A game's cover as a CSS background image, `url("…")`, or none when the game has none. */
export function coverImage(cover: string | undefined): string | undefined {
  return cover === undefined ? undefined : `url(${JSON.stringify(cover)})`;
}

/** The platforms that Exophase lists for a game: "PS4 · PS5". */
export function platformsLabel(platforms: readonly string[]): string {
  return platforms.join(' · ');
}

const countFormat = new Intl.NumberFormat('en-GB');

/** The trophies I earned in a game, with the percentage rounded down, so only all of them make 100%. */
export function trophiesLabel({ earned, total, percent }: Trophies): string {
  const noun = total === 1 ? 'trophy' : 'trophies';
  return `${countFormat.format(earned)} of ${countFormat.format(total)} ${noun} · ${Math.floor(percent)}%`;
}

const minutesPerHour = 60;
const duration = new Intl.DurationFormat('en-GB', { style: 'narrow' });

/** How long I've played a game: "12h 30m", "45m" or "3h". */
export function playtimeLabel(minutes: number): string {
  return duration.format({ hours: Math.floor(minutes / minutesPerHour), minutes: minutes % minutesPerHour });
}
