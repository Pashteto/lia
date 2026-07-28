import { describe, expect, it } from "vitest";
import { intentFromChip, parseDiscoverQuery } from "../discover-intent";
import { moscowHour, rankDiscover } from "../discover-rank";
import type { LiaEvent } from "../types";

function ev(partial: Partial<LiaEvent> & Pick<LiaEvent, "id" | "title" | "startsAt">): LiaEvent {
  return {
    format: "offline",
    status: "published",
    priceType: "paid",
    categories: [{ id: "c1", slug: "lecture", label: "Лекции" }],
    ...partial,
  };
}

// Fixed "now": Wednesday 2026-07-15 12:00 Moscow (UTC+3) = 09:00Z
const NOW = new Date("2026-07-15T09:00:00Z");

describe("moscowHour", () => {
  it("reads Europe/Moscow wall hour", () => {
    expect(moscowHour("2026-07-15T18:30:00+03:00")).toBe(18);
    expect(moscowHour("2026-07-15T15:00:00Z")).toBe(18);
  });
});

describe("rankDiscover", () => {
  const weekendQuiet = ev({
    id: "1",
    title: "Тихая медиация",
    startsAt: "2026-07-18T11:00:00+03:00", // Sat
    categories: [{ id: "m", slug: "mediation", label: "Медиации" }],
    capacity: 8,
  });
  const weekendLoud = ev({
    id: "2",
    title: "Большой фестиваль",
    startsAt: "2026-07-19T14:00:00+03:00", // Sun
    categories: [{ id: "f", slug: "festival", label: "Фестивали" }],
    capacity: 200,
  });
  const freeFar = ev({
    id: "3",
    title: "Бесплатная лекция",
    startsAt: "2026-07-16T19:00:00+03:00",
    priceType: "free",
    venue: { id: "v", name: "Далеко", lat: 56.0, lon: 38.0 },
  });
  const freeNear = ev({
    id: "4",
    title: "Бесплатно у парка",
    startsAt: "2026-07-16T12:00:00+03:00",
    priceType: "free",
    venue: { id: "v2", name: "Рядом", lat: 55.75, lon: 37.62 },
  });
  const evening = ev({
    id: "5",
    title: "Вечерний кинопоказ",
    startsAt: "2026-07-16T19:00:00+03:00", // Thu 19:00
    categories: [{ id: "fi", slug: "film", label: "Кино" }],
  });
  const kids = ev({
    id: "6",
    title: "Мастер-класс для детей",
    startsAt: "2026-07-17T11:00:00+03:00",
    description: "Семейная программа",
  });
  const plain = ev({
    id: "7",
    title: "Обычная лекция",
    startsAt: "2026-07-20T10:00:00+03:00",
  });

  const catalogue = [weekendQuiet, weekendLoud, freeFar, freeNear, evening, kids, plain];

  it("quiet_weekend prefers small mediation and reasons", () => {
    const r = rankDiscover(catalogue, intentFromChip("quiet_weekend"), { now: NOW });
    expect(r.hits.length).toBeGreaterThan(0);
    expect(r.hits.length).toBeLessThanOrEqual(3);
    expect(r.hits[0].event.id).toBe("1");
    expect(r.hits[0].matchReason).toMatch(/группа 8|тихий формат|выходные/);
    expect(r.answer).toMatch(/выходн/i);
  });

  it("free_nearby without geo is city-wide and omits distance claim", () => {
    const r = rankDiscover(catalogue, intentFromChip("free_nearby"), {
      now: NOW,
      userLatLon: null,
    });
    expect(r.omittedDistanceClaim).toBe(true);
    expect(r.answer).toBe("Бесплатные события в городе.");
    expect(r.hits.every((h) => h.event.priceType === "free")).toBe(true);
  });

  it("free_nearby with geo keeps near free events", () => {
    const r = rankDiscover(catalogue, intentFromChip("free_nearby"), {
      now: NOW,
      userLatLon: [55.742, 37.618],
    });
    expect(r.omittedDistanceClaim).toBe(false);
    expect(r.hits.map((h) => h.event.id)).toContain("4");
    expect(r.hits.map((h) => h.event.id)).not.toContain("3");
  });

  it("evening_pair keeps hour>=18 within 7 days", () => {
    const r = rankDiscover(catalogue, intentFromChip("evening_pair"), { now: NOW });
    expect(r.hits.some((h) => h.event.id === "5")).toBe(true);
    expect(r.hits[0].matchReason).toMatch(/вечер/);
  });

  it("with_kids hard-match vs fallback", () => {
    const hard = rankDiscover(catalogue, intentFromChip("with_kids"), { now: NOW });
    expect(hard.usedKidsFallback).toBe(false);
    expect(hard.hits[0].event.id).toBe("6");
    expect(hard.hits[0].matchReason).toBe("семейный формат");

    const weak = rankDiscover([plain], intentFromChip("with_kids"), { now: NOW });
    expect(weak.usedKidsFallback).toBe(true);
    expect(weak.hits[0].matchReason).toBe("ближайшие события");
    expect(weak.answer).toMatch(/уточните/);
  });

  it("caps at 3 and keyword path works", () => {
    const many = Array.from({ length: 5 }, (_, i) =>
      ev({
        id: `k${i}`,
        title: `медиация номер ${i}`,
        startsAt: `2026-07-${16 + i}T12:00:00+03:00`,
      }),
    );
    const r = rankDiscover(many, parseDiscoverQuery("медиация")!, { now: NOW });
    expect(r.hits).toHaveLength(3);
    expect(r.answer).toContain("медиация");
  });

  it("empty catalogue → empty hits + still a sentence", () => {
    const r = rankDiscover([], intentFromChip("evening_pair"), { now: NOW });
    expect(r.hits).toEqual([]);
    expect(r.answer.length).toBeGreaterThan(0);
  });
});
