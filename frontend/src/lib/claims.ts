export function unknownCitations(ids: string[], known: Set<string>): string[] {
  const seen = new Set<string>();
  const missing: string[] = [];
  for (const id of ids) {
    if (known.has(id) || seen.has(id)) continue;
    seen.add(id);
    missing.push(id);
  }
  return missing;
}

export function money(minor: number, currency = "USD"): string {
  const sign = minor < 0 ? "-" : "";
  const abs = Math.abs(minor);
  const major = Math.floor(abs / 100);
  const cents = String(abs % 100).padStart(2, "0");
  return `${sign}${currency} ${major}.${cents}`;
}
