# UI redesign rules for agents

Use this file whenever you redesign FinGuard’s interface: pages, layout, CSS, or components. It is a bias corrector, not a ban list.

Source: [febbhav/signs-of-ai-design](https://github.com/febbhav/signs-of-ai-design), especially `design-rules.md` and Part I (websites and app interfaces). That guide is licensed [CC BY-SA 4.0](https://creativecommons.org/licenses/by-sa/4.0/). One match proves nothing. The fingerprint is many defaults together, with no decision anywhere.

The goal is not to forbid these patterns. Every item below is a legitimate choice in the right context. The rule is: make it a decision, not a default. When one of these fits this product, use it on purpose and say why.

## Core principle

You default to the statistical median of your training data whenever a design choice is left unspecified. That median is the “AI look.” Before shipping any UI, assume your first instinct for color, type, and layout is that median, and make one deliberate departure from it.

Judge the absence of decisions, not the presence of a style. Inter is not a bad typeface. Inter, unchosen, next to twenty other unchosen defaults, is the tell.

## Scope of a redesign

Change how the existing desk looks and how information is weighted. Do not invent a different product.

Do not:

- Turn the case queue or case workspace into a marketing page (hero, logo wall, pricing, FAQ, testimonial carousel).
- Add a fraud score, a guilt label, an account freeze, or any control that moves money. Resolve still means no adverse action.
- Invent user counts, client logos, testimonials, compliance badges, or roadmap claims. This demo has synthetic tenants and stated limits. Leave trust chrome out until the content is real.
- Remove evidence citations, the audit trail, role differences, tenant isolation, or the states the desk already has: loading, empty queue, version conflict, report failed or rejected, viewer read-only.
- Restyle by swapping tokens on the same equal card grid. Signals, evidence, and the human decision have to stay visually primary.

Routes stay `/`, `/login`, `/workspace`, and `/cases/:id`.

## Color

- Do not default to an indigo or purple gradient (the Tailwind `indigo-500` / `violet-500` / `purple-500` reflex, or `bg-gradient-to-br from-blue-600 to-purple-500`). Pick from the product. FinGuard’s committed accent is the navy already in `theme.css` (`#16324f`), with amber and rose only for warning and danger. If you change the palette, choose one dominant color and one accent, and say why.
- Do not fall back to emerald (`#10B981`) or a cream page (`#F4F1EA` / `#faf8f4`) just because purple is off the table. Those are the second-wave defaults.
- Do not use neon-on-near-black with colored glow shadows (`box-shadow: 0 0 20px rgba(139,92,246,0.5)`). This is a light investigation desk, not a dark developer landing page.
- Do not use four muted colors at equal weight. One color leads. Success, warning, and danger stay semantic.
- Do not fill headings or big numbers with gradient text (`bg-clip-text`).
- Do not float purple or violet gradient orbs or blobs behind a hero.

## Typography

- Inter, Geist, and Space Grotesk are fine typefaces, and they are the defaults everyone ships. `styles.css` still names Inter while `theme.css` overrides to Segoe UI. Do not leave that split as an accident. Choose one stack for the product and state the choice. If you keep a system sans, that is a decision.
- Do not reach for an oversized italic serif hero (Instrument Serif, Playfair Display, and the same idea in Georgia) unless a specific editorial moment needs it. `landing.css` and `refinement.css` already drop Georgia in places. Do not spread that into the case desk.
- Build a real type scale with clear steps. Do not set the whole page in one family at one weight, with size steps too close to create hierarchy.
- Do not crush letter spacing on display type.
- Do not spray all-caps section kickers above every heading (`FEATURES`, `SIGNALS`, `NOTES`).
- Do not set body copy or ordinary labels in a monospace font for atmosphere. Monospace stays on identifiers, evidence ids, and code-like values.
- Pick one case convention for headings, buttons, and labels. Do not Title Case everything by reflex.

## Layout

- Do not emit the default hero: centered eyebrow pill (“New”, “v2.0”, a sparkle prefix), oversized centered headline, gray one-line subhead, two buttons (gradient primary reading “Get Started” plus a ghost secondary), gradient orb behind. The landing page may stay a landing page. If you rebuild that hero, vary the formula. The workspace is not a hero.
- Do not resolve every section into exactly three equal cards with an icon on top, a short heading, and two lines of text. Let the content decide the count and shape. The landing preview has three real product shots. Do not copy that rhythm into the case file.
- A bento grid is the same reflex once three cards are banned. It does not count as a departure.
- Do not stamp numbered markers (`01` / `02` / `03`) on sections that are not a sequence.
- Do not ship the template waterfall in fixed order: hero, logo wall, three cards, testimonials, stats row, pricing, FAQ, footer call to action. Cut or reorder to fit this product.
- Vary spacing. Do not use one section padding from top to bottom, and do not express all spacing as `gap-4` / `p-6` / `my-8`.
- On the case desk, do not give the primary action and a tertiary link the same visual weight. One thing on the screen leads: the open case and the next human decision.

## Components and surfaces

- Do not apply one large radius (`rounded-2xl`, about 16–24px) to every card, input, button, and image. Vary radius by element. Buttons in this app are already near 7px. Do not balloon them into pills.
- If you introduce a component library (shadcn or similar), override the default theme tokens. Shipping untouched `rounded-2xl shadow-lg p-6`, zinc or slate, and the default button is the tell.
- Do not pair a hairline border with a wide diffuse shadow on the same card. Commit to one edge treatment. This desk currently uses a 1px line and no card shadow. Keep that commitment or replace it on purpose.
- Do not put a decorative colored strip on the left edge of ordinary cards. A side accent belongs on semantic alerts and on the active nav item, where it already marks state.
- Do not nest cards inside cards. Containment is not hierarchy.
- Do not add frosted glass (`backdrop-blur`) or glow borders where nothing is layered underneath.
- Do not animate layout properties. Transitions in this app are background and border color, about 160ms. Keep motion that short and coherent.

## Icons, images, and motion

- Do not reuse the same five Lucide icons for unrelated ideas: Sparkles for AI, Zap for fast, Shield for secure, Check for benefits, BarChart3 for analytics, ArrowRight on every action. Pick an icon that matches the specific meaning, or use none.
- Never use emoji as feature-card or section icons.
- Do not use a fake terminal window with macOS traffic-light dots as a hero visual. This is not a developer-tool marketing page.
- Do not replace the real workspace screenshots with plastic 3D blobs, hand-coded SVG scenes, or generated headshots. If you add images, avoid the warm yellow cast, waxy poreless skin, garbled background text, and hands under load. Prefer the existing local photography and the actual product screenshots.
- Do not apply the same fade-up-on-scroll (`opacity: 0`, `y: 20`) to every section.
- Do not use bounce or elastic easing. Do not mix snap buttons with elaborate card motion.
- One orchestrated moment is enough. Motion that answers an action (open, expand, confirm) is welcome when it shows what changed.

## Copy

- Do not write a headline that could describe any product (“Build faster. Ship smarter.”, “Elevate your workflow”, “Operational clarity without the clutter”). Say what this screen does: review a case, read a signal, record a decision.
- Strip the buzzword layer: streamline, supercharge, empower, world-class, enterprise-grade, seamless, robust, unlock, unleash, “in today’s fast-paced world.”
- Do not overuse the em dash, the “it’s not just X, it’s Y” pivot, or a manufactured-contrast line at the end of every section.
- Do not leave generator attribution in the UI (“Built with v0”, “Built on Lovable”).
- Buttons name the action (`Start review`, `Resolve`). Do not relabel them “Get Started.”

## The one rule

No single item above matters on its own. A rounded card or a sans-serif font is fine. The AI look is many of these defaults at once, with no deliberate choice anywhere. Ship a design where the important decisions were made on purpose.

Before you finish a visual pass, check the screen against this list. If several items fired and none of them was a stated choice for FinGuard, change those parts and say what you changed.
