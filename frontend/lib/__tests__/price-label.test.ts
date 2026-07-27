import { describe, expect, it } from 'vitest';
import { priceLabel } from '../price-label';

describe('priceLabel', () => {
  it('free is the literal FREE', () => {
    expect(priceLabel(0)).toBe('FREE');
    expect(priceLabel(null)).toBe('FREE');
    expect(priceLabel(undefined)).toBe('FREE');
    expect(priceLabel(500, 'free')).toBe('FREE');
  });
  it('formats rubles with nbsp grouping', () => {
    expect(priceLabel(800)).toBe('800 ₽');
    expect(priceLabel(1500)).toBe(`1 500 ₽`);
  });
  it('from prices get от prefix', () => {
    expect(priceLabel(1500, 'from')).toBe(`от 1 500 ₽`);
  });
});
