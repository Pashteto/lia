// Soobscha intro splash — letters gather into the word, then the word settles into the site header.
const { useComposition, Easing, animate, Captions } = window;

const MOTION = {
  enter: Easing.easeOutCubic,
  gather: Easing.easeInOutQuart,
  settle: Easing.easeInOutCubic,
};

const W = 1440, H = 900;
const INK = '#111', PAPER = '#F2F0EC', MUTED = '#8A857C';
const LETTERS = ['с', 'о', 'о', 'б', 'щ', 'а'];
// scattered start offsets (px) and rotations (deg) — one per letter
const SCATTER = [
  { x: -560, y: -300, r: -18 },
  { x: -260, y: 330, r: 12 },
  { x: 80, y: -380, r: 22 },
  { x: 380, y: 300, r: -14 },
  { x: 620, y: -200, r: 9 },
  { x: -140, y: 380, r: -24 },
];

function SoobschaIntro() {
  const { T, CUES, authoredTotal } = useComposition();

  // per-letter entrance + gather
  const letterStyle = (i) => {
    const s = SCATTER[i];
    const stagger = i * 0.05;
    const op = animate({ from: 0, to: 1, start: CUES.Scatter + stagger, end: CUES.Scatter + stagger + 0.3, ease: MOTION.enter })(T);
    const k = animate({ from: 1, to: 0, start: CUES.Gather + stagger * 0.5, end: CUES.Gather + 0.75 + stagger * 0.5, ease: MOTION.gather })(T);
    return {
      display: 'inline-block',
      opacity: op,
      transform: `translate(${s.x * k}px, ${s.y * k}px) rotate(${s.r * k}deg)`,
      willChange: 'transform',
    };
  };

  // word group: center → header
  const p = animate({ from: 0, to: 1, start: CUES.Open, end: CUES.Open + 0.7, ease: MOTION.settle })(T);
  const fs = 220 + (15 - 220) * p;
  const left = W / 2 + (48 - W / 2) * p;
  const top = H / 2 - 40 + (30 - (H / 2 - 40)) * p;
  const k = 1 - p;
  const ls = -0.06 + (-0.04 + 0.06) * p;

  // signature (rule + address) under the word
  const ruleW = animate({ from: 0, to: 1, start: CUES.Sign, end: CUES.Sign + 0.4, ease: MOTION.enter })(T);
  const addrOp = animate({ from: 0, to: 1, start: CUES.Sign + 0.2, end: CUES.Sign + 0.45, ease: MOTION.enter })(T);
  const signOut = animate({ from: 1, to: 0, start: CUES.Open, end: CUES.Open + 0.25, ease: MOTION.settle })(T);

  // header chrome + page content
  const chrome = animate({ from: 0, to: 1, start: CUES.Open + 0.5, end: CUES.Open + 0.85, ease: MOTION.enter })(T);
  const body = animate({ from: 0, to: 1, start: CUES.Open + 0.7, end: CUES.Open + 1.15, ease: MOTION.enter })(T);
  const bodyY = (1 - body) * 18;

  const nav = ['События', 'Подбор', 'Календарь', 'Карта', 'Организаторам', 'Войти'];

  return (
    <div style={{ position: 'absolute', inset: 0, background: PAPER, color: INK, fontFamily: "'Archivo', sans-serif", overflow: 'hidden' }}>
      {/* header rule */}
      <div style={{ position: 'absolute', left: 0, right: 0, top: 68, height: 1, background: INK, transform: `scaleX(${chrome})`, transformOrigin: 'left' }} />
      {/* nav */}
      <div style={{ position: 'absolute', right: 48, top: 40, display: 'flex', gap: 22, opacity: chrome, fontSize: 11, letterSpacing: '.14em', textTransform: 'uppercase', fontFamily: "'Space Grotesk', sans-serif" }}>
        {nav.map((n, i) => (
          <span key={n} style={{ borderBottom: i === 0 ? `2px solid ${INK}` : '2px solid transparent', paddingBottom: 2, fontWeight: i === 0 ? 700 : 400 }}>{n}</span>
        ))}
      </div>

      {/* wordmark — one element travelling from center to header */}
      <div style={{ position: 'absolute', left, top, transform: `translate(calc(-50% * ${k}), calc(-50% * ${k}))`, whiteSpace: 'nowrap', lineHeight: 1 }}>
        <div style={{ fontSize: fs, fontWeight: 900, letterSpacing: `${ls}em`, display: 'flex' }}>
          {LETTERS.map((ch, i) => <span key={i} style={letterStyle(i)}>{ch}</span>)}
        </div>
        <div style={{ opacity: signOut, marginTop: 22 * k, height: 40 * k, overflow: 'hidden' }}>
          <div style={{ height: 2, background: INK, transform: `scaleX(${ruleW})`, transformOrigin: 'left' }} />
          <div style={{ marginTop: 12, display: 'flex', justifyContent: 'space-between', opacity: addrOp, fontFamily: "'JetBrains Mono', monospace", fontSize: 14, letterSpacing: '.18em', color: MUTED }}>
            <span>SOOBSCHA.RU</span><span>МОСКВА · 2026</span>
          </div>
        </div>
      </div>

      {/* page body */}
      <div style={{ position: 'absolute', left: 48, right: 48, top: 110, opacity: body, transform: `translateY(${bodyY}px)` }}>
        <div style={{ fontFamily: "'Space Grotesk', sans-serif", fontSize: 11, letterSpacing: '.13em', textTransform: 'uppercase', color: MUTED, marginBottom: 14 }}>Москва · 42 события · Сентябрь 2026</div>
        <div style={{ fontSize: 72, fontWeight: 900, letterSpacing: '-.03em', lineHeight: .94, whiteSpace: 'nowrap' }}>Медиации, лекции<br />и разговоры об искусстве</div>
        <div style={{ marginTop: 40, display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', borderTop: `1px solid ${INK}` }}>
          {[['01', 'Фестивали', 'Летний фестиваль медиаискусства', 'Музей «Гараж»', '25–26.09'],
            ['02', 'Медиации', 'Медиация в залах старых мастеров', 'ГМИИ им. Пушкина', '28.09'],
            ['03', 'Лекции', 'Лаборатория медиаций', 'Парк Горького', '02.10']].map((c, i) => (
            <div key={c[0]} style={{ padding: '16px 18px', borderRight: i < 2 ? `1px solid ${INK}` : 'none', borderBottom: `1px solid ${INK}`, display: 'flex', flexDirection: 'column', gap: 8, minHeight: 150 }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', fontFamily: "'JetBrains Mono', monospace", fontSize: 13 }}><b>{c[0]}</b><span style={{ fontFamily: "'Space Grotesk', sans-serif", fontSize: 10, letterSpacing: '.13em', textTransform: 'uppercase', color: MUTED }}>{c[1]}</span></div>
              <div style={{ fontSize: 20, fontWeight: 900, letterSpacing: '-.02em', lineHeight: 1.02 }}>{c[2]}</div>
              <div style={{ fontFamily: "'Space Grotesk', sans-serif", fontSize: 10, letterSpacing: '.13em', textTransform: 'uppercase', color: MUTED }}>{c[3]}</div>
              <div style={{ marginTop: 'auto', display: 'flex', justifyContent: 'space-between', alignItems: 'baseline' }}><span style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 13 }}>{c[4]}</span><span style={{ fontWeight: 900, fontSize: 14 }}>FREE</span></div>
            </div>
          ))}
        </div>
      </div>

      <Captions items={[
        { at: CUES.Gather + 0.2, until: CUES.Sign, text: 'вместе · общими усилиями' },
      ]} style={{ bottom: 72, font: "400 12px 'Space Grotesk', sans-serif", letterSpacing: '.24em', textTransform: 'uppercase', color: MUTED, textShadow: 'none' }} />
    </div>
  );
}

window.SoobschaIntro = SoobschaIntro;
