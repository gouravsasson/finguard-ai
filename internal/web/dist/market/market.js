const params = new URLSearchParams(location.search);
const reduced = window.matchMedia("(prefers-reduced-motion: reduce)").matches || params.has("still");
const desktop = () => window.matchMedia("(min-width: 1000px)").matches;

document.body.classList.remove("is-enter");
requestAnimationFrame(() => document.body.classList.add("is-in"));

const nav = document.getElementById("nav");
const burger = document.getElementById("burger");
let lastY = window.scrollY;
let hidden = false;

function onScroll() {
  const y = window.scrollY;
  const atTop = y < 8;
  nav.classList.toggle("is-top", atTop);
  nav.classList.toggle("is-frost", !atTop);

  if (!atTop && y > lastY + 6 && y > 80) hidden = true;
  else if (y < lastY - 6 || atTop) hidden = false;
  nav.classList.toggle("is-hidden", hidden && !nav.classList.contains("is-open"));
  lastY = y;

  const chapters = [...document.querySelectorAll(".chapter")];
  const side = document.getElementById("side");
  const probe = window.innerHeight * 0.45;
  let active = null;
  let any = false;
  chapters.forEach((ch) => {
    const r = ch.getBoundingClientRect();
    if (r.top < probe && r.bottom > probe) {
      any = true;
      active = ch.id;
      nav.classList.toggle("is-dark", ch.dataset.theme === "dark");
    }
  });
  if (!any) nav.classList.remove("is-dark");
  const footer = document.getElementById("footer").getBoundingClientRect();
  const hero = document.getElementById("hero").getBoundingClientRect();
  const inProducts = any && footer.top > window.innerHeight * 0.4 && hero.bottom < window.innerHeight * 0.2;
  side.classList.toggle("is-on", inProducts && desktop());
  side.querySelectorAll("a").forEach((a) => {
    a.classList.toggle("is-active", a.getAttribute("href") === "#" + active);
  });

  if (desktop() && !reduced) {
    document.querySelectorAll("[data-stack]").forEach((stack) => {
      const sticky = stack.querySelector(".stack-sticky");
      const cards = [...sticky.querySelectorAll(".card")];
      const start = stack.getBoundingClientRect().top + window.scrollY;
      const range = stack.offsetHeight - window.innerHeight;
      const p = Math.min(1, Math.max(0, (window.scrollY - start) / Math.max(1, range)));
      const steps = Math.max(1, cards.length - 1);
      cards.forEach((card, i) => {
        const local = p * steps - (i - 1);
        const t = i === 0 ? 1 : Math.min(1, Math.max(0, local));
        const yShift = i === 0 ? 0 : (1 - t) * 140;
        card.style.transform = `translateY(${yShift}%)`;
        card.style.zIndex = String(i + 1);
        const media = card.querySelector(".card-media");
        const scale = 0.96 + 0.04 * (i === 0 ? 1 : t);
        const opacity = i === 0 ? 1 : 0.2 + 0.8 * t;
        media.style.transform = `scale(${scale})`;
        media.style.opacity = String(opacity);
      });
    });
  }
}

window.addEventListener("scroll", onScroll, { passive: true });
onScroll();

burger.addEventListener("click", () => {
  const open = nav.classList.toggle("is-open");
  document.getElementById("sheet").classList.toggle("is-open", open);
  burger.setAttribute("aria-expanded", open ? "true" : "false");
  if (open) nav.classList.remove("is-hidden");
});
document.getElementById("sheet").addEventListener("click", (e) => {
  if (e.target.closest("a")) {
    nav.classList.remove("is-open");
    document.getElementById("sheet").classList.remove("is-open");
    burger.setAttribute("aria-expanded", "false");
  }
});

const marks = [
  ["queue", "#d7e4ff"],
  ["signals", "#f8d1c9"],
  ["evidence", "#ddccff"],
  ["decisions", "#c8f5d4"],
  ["audit", "#ffe7a8"],
  ["records", "#c9e4ff"],
  ["roles", "#e4d4ff"],
  ["replay", "#c8f0ea"],
];
const track = document.getElementById("marqueeTrack");
function paintMarks() {
  const row = document.createElement("div");
  row.style.display = "flex";
  marks.forEach(([name, color]) => {
    const el = document.createElement("div");
    el.className = "mark";
    el.style.background = color;
    el.textContent = name;
    row.appendChild(el);
  });
  return row;
}
track.appendChild(paintMarks());
track.appendChild(paintMarks());

const marquee = document.getElementById("marquee");
let offset = 0;
let paused = false;
let dragging = false;
let dragX = 0;
let width = 0;
function measure() {
  width = track.scrollWidth / 2;
}
measure();
window.addEventListener("resize", measure);

function loop() {
  if (!reduced && !paused && !dragging) offset -= 0.6;
  if (width) {
    if (offset <= -width) offset += width;
    if (offset > 0) offset -= width;
  }
  track.style.transform = `translate3d(${offset}px,0,0)`;
  requestAnimationFrame(loop);
}
if (!reduced) requestAnimationFrame(loop);
marquee.addEventListener("mouseenter", () => { paused = true; });
marquee.addEventListener("mouseleave", () => { paused = false; dragging = false; marquee.classList.remove("is-drag"); });
marquee.addEventListener("pointerdown", (e) => {
  dragging = true;
  dragX = e.clientX;
  marquee.classList.add("is-drag");
  marquee.setPointerCapture(e.pointerId);
});
marquee.addEventListener("pointermove", (e) => {
  if (!dragging) return;
  offset += e.clientX - dragX;
  dragX = e.clientX;
});
marquee.addEventListener("pointerup", () => {
  dragging = false;
  marquee.classList.remove("is-drag");
});

document.querySelectorAll(".faq-q").forEach((btn) => {
  btn.addEventListener("click", () => {
    const item = btn.parentElement;
    const open = item.classList.contains("is-open");
    document.querySelectorAll(".faq-item").forEach((el) => el.classList.remove("is-open"));
    if (!open) item.classList.add("is-open");
  });
});

const footer = document.getElementById("footer");
const io = new IntersectionObserver((entries) => {
  entries.forEach((en) => {
    if (en.isIntersecting) footer.classList.add("is-in");
  });
}, { threshold: 0.25 });
io.observe(footer);

const cookie = document.getElementById("cookie");
function dismissCookie() {
  cookie.classList.remove("is-in");
  cookie.addEventListener("transitionend", () => cookie.classList.add("is-gone"), { once: true });
  if (reduced) cookie.classList.add("is-gone");
}
if (params.has("bare")) cookie.classList.add("is-gone");
else if (reduced) cookie.classList.add("is-in");
else requestAnimationFrame(() => cookie.classList.add("is-in"));

const at = params.get("at");
if (at) {
  const go = () => {
    if (at.startsWith("#")) {
      const el = document.querySelector(at);
      if (el) window.scrollTo(0, el.getBoundingClientRect().top + window.scrollY - 20);
    } else window.scrollTo(0, Number(at) || 0);
    onScroll();
  };
  go();
  window.addEventListener("load", go);
}
document.getElementById("cookieAccept").addEventListener("click", dismissCookie);
document.getElementById("cookieReject").addEventListener("click", dismissCookie);
