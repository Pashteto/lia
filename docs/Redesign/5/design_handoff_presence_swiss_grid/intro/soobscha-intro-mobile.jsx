// Soobscha intro splash — mobile 390×844. Same four beats as the desktop piece.
const { useComposition, Easing, animate, Captions } = window;

const MOTION = {
  enter: Easing.easeOutCubic,
  gather: Easing.easeInOutQuart,
  settle: Easing.easeInOutCubic,
};

const W = 390, H = 844;
const INK = '#111', PAPER = '#F2F0EC', MUTED = '#8A857C';
const LETTERS = ['с', 'о', 'о', 'б', 'щ', 'а'];
const SCATTER = [
  { x: -150, y: -300, r: -18 },
  { x: -110, y: 280, r: 12 },
  { x: 40, y: -360, r: 22 },
  { x: 150, y: 260, r: -14 },
  { x: 160, y: -180, r: 9 },
  { x: -40, y: 340, r: -24 },
];

function SoobschaIntroMobile() {
  const { T, CUES } = useComposition();

  const letterStyle = (i) => {
    const s = SCATTER[i];
    const stagger = i * 0.05;
    const op = animate({ from: 0, to: 1, start: CUES.Scatter + stagger, end: CUES.Scatter + stagger + 0.3, ease: MOTION.enter })(T);
    const k = animate({ from: 1, to: 0, start: CUES.Gather + stagger * 0.5, end: CUES.Gather + 0.75 + stagger * 0.5, ease: MOTION.gather })(T);
    return { display: 'inline-block', opacity: op, transform: `translate(${s.x * k}px, ${s.y * k}px) rotate(${s.r * k}deg)`, willChange: 'transform' };
  };

  const p = animate({ from: 0, to: 1, start: CUES.Open, end: CUES.Open + 0.7, ease: MOTION.settle })(T);
  const fs = 84 + (17 - 84) * p;
  const left = W / 2 + (18 - W / 2) * p;
  const top = H / 2 - 30 + (58 - (H / 2 - 30)) * p;
  const k = 1 - p;
  const ls = -0.06 + (-0.04 + 0.06) * p;

  const ruleW = animate({ from: 0, to: 1, start: CUES.Sign, end: CUES.Sign + 0.4, ease: MOTION.enter })(T);
  const addrOp = animate({ from: 0, to: 1, start: CUES.Sign + 0.2, end: CUES.Sign + 0.45, ease: MOTION.enter })(T);
  const signOut = animate({ from: 1, to: 0, start: CUES.Open, end: CUES.Open + 0.25, ease: MOTION.settle })(T);

  const chrome = animate({ from: 0, to: 1, start: CUES.Open + 0.5, end: CUES.Open + 0.85, ease: MOTION.enter })(T);
  const body = animate({ from: 0, to: 1, start: CUES.Open + 0.7, end: CUES.Open + 1.15, ease: MOTION.enter })(T);
  const bodyY = (1 - body) * 14;

  const rows = [
    ['01', 'Летний фестиваль медиаискусства', 'Гараж · 25.09', 'FREE'],
    ['02', 'Медиация в залах старых мастеров', 'ГМИИ · 28.09', 'FREE'],
    ['03', 'Лаборатория медиаций', 'Парк Горького · 02.10', 'FREE'],
    ['04', 'Кинопоказ с обсуждением', 'Еврейский музей · 04.10', 'FREE'],
  ];
  const tabs = ['Лента', 'Подбор', 'Карта', 'Я'];
  const cap = { fontFamily: "'Space Grotesk', sans-serif", fontSize: 9, letterSpacing: '.13em', textTransform: 'uppercase', color: MUTED };

  return (
    <div style={{ position: 'absolute', inset: 0, background: PAPER, color: INK, fontFamily: "'Archivo', sans-serif", overflow: 'hidden' }}>
      {/* status bar spacer + header */}
      <div style={{ position: 'absolute', left: 0, right: 0, top: 86, height: 1, background: INK, transform: `scaleX(${chrome})`, transformOrigin: 'left' }} />
      <div style={{ position: 'absolute', right: 18, top: 58, opacity: chrome, ...cap }}>МСК · 42</div>

      {/* wordmark */}
      <div style={{ position: 'absolute', left, top, transform: `translate(calc(-50% * ${k}), calc(-50% * ${k}))`, whiteSpace: 'nowrap', lineHeight: 1 }}>
        <div style={{ fontSize: fs, fontWeight: 900, letterSpacing: `${ls}em`, display: 'flex' }}>
          {LETTERS.map((ch, i) => <span key={i} style={letterStyle(i)}>{ch}</span>)}
        </div>
        <div style={{ opacity: signOut, marginTop: 16 * k, height: 34 * k, overflow: 'hidden' }}>
          <div style={{ height: 2, background: INK, transform: `scaleX(${ruleW})`, transformOrigin: 'left' }} />
          <div style={{ marginTop: 10, display: 'flex', justifyContent: 'space-between', opacity: addrOp, fontFamily: "'JetBrains Mono', monospace", fontSize: 11, letterSpacing: '.16em', color: MUTED }}>
            <span>SOOBSCHA.RU</span><span>МСК · 2026</span>
          </div>
        </div>
      </div>

      {/* body */}
      <div style={{ position: 'absolute', left: 0, right: 0, top: 87, bottom: 72, opacity: body, transform: `translateY(${bodyY}px)`, display: 'flex', flexDirection: 'column' }}>
        <div style={{ padding: '18px 18px 14px', borderBottom: `1px solid ${INK}` }}>
          <div style={{ fontSize: 30, fontWeight: 900, letterSpacing: '-.03em', lineHeight: 1.02 }}>Что смотреть<br />на неделе</div>
        </div>
        <div style={{ padding: '10px 18px', display: 'flex', gap: 6, borderBottom: `1px solid ${INK}` }}>
          {['Все', 'Сегодня', 'Выходные', 'Бесплатно'].map((c, i) => (
            <span key={c} style={{ border: `1px solid ${INK}`, padding: '5px 9px', fontSize: 9, letterSpacing: '.12em', textTransform: 'uppercase', background: i === 0 ? INK : 'transparent', color: i === 0 ? '#fff' : INK, fontFamily: "'Space Grotesk', sans-serif" }}>{c}</span>
          ))}
        </div>
        {rows.map((r) => (
          <div key={r[0]} style={{ padding: '13px 18px', borderBottom: '1px solid #ddd', display: 'grid', gridTemplateColumns: '26px 1fr auto', gap: '3px 10px', alignItems: 'baseline' }}>
            <span style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 12, fontWeight: 700 }}>{r[0]}</span>
            <span style={{ fontSize: 15, fontWeight: 700, lineHeight: 1.1 }}>{r[1]}</span>
            <span style={{ fontSize: 12, fontWeight: 900 }}>{r[3]}</span>
            <span /><span style={cap}>{r[2]}</span><span />
          </div>
        ))}
      </div>

      {/* tab bar */}
      <div style={{ position: 'absolute', left: 0, right: 0, bottom: 0, height: 72, borderTop: `1px solid ${INK}`, background: PAPER, display: 'flex', opacity: chrome, paddingBottom: 14, boxSizing: 'border-box' }}>
        {tabs.map((t, i) => (
          <div key={t} style={{ flex: 1, display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', gap: 5, color: i === 0 ? INK : MUTED }}>
            <div style={{ width: 16, height: 16, border: '1.5px solid currentColor', background: i === 0 ? INK : 'transparent', boxSizing: 'border-box' }} />
            <span style={{ fontFamily: "'Space Grotesk', sans-serif", fontSize: 8, letterSpacing: '.1em', textTransform: 'uppercase', fontWeight: i === 0 ? 700 : 400 }}>{t}</span>
          </div>
        ))}
      </div>

      <Captions items={[{ at: CUES.Gather + 0.2, until: CUES.Sign, text: 'вместе · общими усилиями' }]}
        style={{ bottom: 110, left: '6%', right: '6%', font: "400 10px 'Space Grotesk', sans-serif", letterSpacing: '.22em', textTransform: 'uppercase', color: MUTED, textShadow: 'none' }} />
    </div>
  );
}

window.SoobschaIntroMobile = SoobschaIntroMobile;
