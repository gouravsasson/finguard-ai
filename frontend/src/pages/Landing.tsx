import { useState } from "react";
import { Link } from "react-router-dom";
import { Brand, Icon } from "../components/Icon";
import "../landing.css";

export function Landing({ signedIn }: { signedIn: boolean }) {
  const [menuOpen, setMenuOpen] = useState(false);
  const [preview, setPreview] = useState(0);
  const previews = [
    {
      title: "Case queue",
      label: "Queue",
      image: "workspace-preview.png",
      alt: "FinGuard case queue with priorities, review status, and account search",
      description:
        "Keep your team's open work in view. Search accounts, filter priorities, and move directly into an investigation.",
    },
    {
      title: "Investigation",
      label: "Case file",
      image: "case-preview.png",
      alt: "FinGuard case investigation with source evidence, detected signals, and transaction details",
      description:
        "Read each signal alongside its source records. Keep the account, transactions, and evidence in the same workspace.",
    },
    {
      title: "Relationships",
      label: "Links",
      image: "relationships-preview.png",
      alt: "FinGuard observed relationship graph connecting accounts, transactions, merchants, devices, and locations",
      description:
        "Explore the relationships recorded in the data. Observed connections provide context, not conclusions about identity.",
    },
  ];
  const entry = signedIn ? "/workspace" : "/login";
  return (
    <div className="landing">
      <header className="site-header">
        <Link to="/" className="brand-link" aria-label="FinGuard home">
          <Brand />
        </Link>
        <button
          className="menu-toggle"
          aria-expanded={menuOpen}
          aria-controls="site-navigation"
          onClick={() => setMenuOpen(!menuOpen)}
        >
          {menuOpen ? "Close" : "Menu"}
          <span>{menuOpen ? "−" : "+"}</span>
        </button>
        <nav
          id="site-navigation"
          className={menuOpen ? "site-navigation is-open" : "site-navigation"}
          aria-label="Main navigation"
        >
          <a href="#platform" onClick={() => setMenuOpen(false)}>
            The platform
          </a>
          <a href="#workflow" onClick={() => setMenuOpen(false)}>
            How it works
          </a>
          <a href="#principles" onClick={() => setMenuOpen(false)}>
            Our approach
          </a>
          <Link className="nav-signin" to={entry}>
            {signedIn ? "Your workspace" : "Sign in"}
            <Icon name="arrow" size={16} />
          </Link>
        </nav>
      </header>
      <main id="content">
        <section className="hero">
          <div className="hero-copy">
            <p className="editorial-label">Northwind desk · synthetic records</p>
            <h1>Every signal stays next to the transaction that produced it.</h1>
            <p>
              Open a case, read the rule threshold, and check the source record
              before anyone writes a decision. The desk does not freeze
              accounts or move money.
            </p>
            <div className="hero-actions">
              <Link className="site-button" to={entry}>
                Explore the workspace
                <Icon name="arrow" size={18} />
              </Link>
              <a className="text-link" href="#platform">
                Take a closer look <span>↗</span>
              </a>
            </div>
            <div className="hero-footnote">
              <span className="tiny-square" /> Interactive demo · Synthetic data
              · No financial actions
            </div>
          </div>
          <figure className="hero-photo">
            <img
              src="/images/workspace-preview.png"
              alt="FinGuard case queue with priorities, review status, and account search"
              width="1440"
              height="1000"
              fetchPriority="high"
            />
            <figcaption>The case queue. Synthetic records only.</figcaption>
          </figure>
        </section>
        <div className="principle-strip">
          <span>On this desk</span>
          <p>Traceable evidence</p>
          <i />
          <p>Explainable signals</p>
          <i />
          <p>Human decisions</p>
        </div>
        <section id="platform" className="platform-section site-section">
          <div className="section-intro">
            <div>
              <p className="editorial-label">Workspace</p>
              <h2>Queue, case file, and source records on one desk.</h2>
            </div>
            <p>
              From the first signal to the final note, keep the investigation in
              one place. Follow each finding back to its source and build a
              record others can review.
            </p>
          </div>
          <div className="preview-switcher" aria-label="Workspace preview">
            {previews.map((item, index) => (
              <button
                type="button"
                key={item.title}
                aria-pressed={preview === index}
                className={preview === index ? "selected" : ""}
                onClick={() => setPreview(index)}
              >
                <span>{item.label}</span>
                {item.title}
                <Icon name="arrow" size={17} />
              </button>
            ))}
          </div>
          <div className="product-stage">
            <div className="product-frame">
              <div className="product-frame-bar">
                <span>
                  <i />
                  <i />
                  <i />
                </span>
                <span>FinGuard / Investigation workspace</span>
                <span className="preview-label">SYNTHETIC DEMO</span>
              </div>
              <img
                src={`/images/${previews[preview].image}`}
                alt={previews[preview].alt}
                width="1440"
                height="1000"
                loading="lazy"
              />
            </div>
          </div>
          <div className="product-caption">
            <p>{previews[preview].description}</p>
            <Link to={entry}>
              Open the demo
              <Icon name="arrow" size={17} />
            </Link>
          </div>
        </section>
        <section id="workflow" className="workflow-section site-section">
          <div className="section-intro">
            <div>
              <p className="editorial-label">How a case moves</p>
              <h2>Read the signal, check the record, then write the decision.</h2>
            </div>
            <p>
              Rules surface the activity. Evidence provides the context. Your
              team decides what it means.
            </p>
          </div>
          <div className="workflow-steps">
            {[
              {
                n: "01",
                title: "Find the starting point",
                text: "Review versioned signals with their thresholds and observed values. Filter the queue to focus on the cases that need your attention.",
                icon: "search",
              },
              {
                n: "02",
                title: "Follow the evidence",
                text: "Inspect transactions and source records. Use the knockout desk to see which hypothetical edit would clear a signal, and which would leave it in place.",
                icon: "folder",
              },
              {
                n: "03",
                title: "Record your judgment",
                text: "Add notes, escalate for review, or resolve with a rationale. The audit trail records who acted and why.",
                icon: "check",
              },
            ].map((step) => (
              <article key={step.n}>
                <div className="step-top">
                  <span>{step.n}</span>
                  <Icon
                    name={step.icon as "search" | "folder" | "check"}
                    size={23}
                  />
                </div>
                <h3>{step.title}</h3>
                <p>{step.text}</p>
              </article>
            ))}
          </div>
        </section>
        <section id="principles" className="principles-section site-section">
          <div className="principles-copy">
            <p className="editorial-label">Limits</p>
            <h2>The model drafts. A person decides.</h2>
            <p>
              Useful investigation software should make its reasoning easier to
              examine. That principle shapes how FinGuard handles signals, model
              output, and decisions.
            </p>
            <p className="principles-note">
              Resolution on this desk is recorded as no adverse action.
            </p>
          </div>
          <div className="principle-details">
            <details open>
              <summary>
                <span>01</span>Every finding needs a source<i>+</i>
              </summary>
              <p>
                Report claims link to supplied evidence IDs. Unknown references
                and unsupported claims are validated by the backend rather than
                presented as established facts.
              </p>
            </details>
            <details>
              <summary>
                <span>02</span>AI has a defined role<i>+</i>
              </summary>
              <p>
                The assistant helps review case evidence and identify gaps. It
                cannot freeze accounts, move money, or make decisions. Model
                mode is visible, including when the local demonstration stub is
                used.
              </p>
            </details>
            <details>
              <summary>
                <span>03</span>Decisions belong to people<i>+</i>
              </summary>
              <p>
                Role permissions control who can investigate, escalate, resolve,
                or close a case. Significant decisions require a rationale and
                are recorded in the audit trail.
              </p>
            </details>
            <details>
              <summary>
                <span>04</span>Knock a signal out before you decide<i>+</i>
              </summary>
              <p>
                The knockout desk replays the same deterministic rules under a
                hypothetical edit. One replay may clear the signal. Another may
                show that dropping a single transaction is not enough. Nothing
                in the ledger changes, and a cleared replay is not proof the
                activity was legitimate.
              </p>
            </details>
            <details>
              <summary>
                <span>05</span>Clear about the limits<i>+</i>
              </summary>
              <p>
                This is a synthetic-data demonstrator, not a production banking
                system. Triage bands are not fraud probabilities. Shared devices
                and observed relationships do not establish identity.
              </p>
            </details>
          </div>
        </section>
        <section className="closing-section">
          <div>
            <h2>Open the queue and start with acct_demo.</h2>
          </div>
          <div>
            <Link className="site-button" to={entry}>
              Enter the demo workspace
              <Icon name="arrow" size={18} />
            </Link>
            <p>Demo access is included. No real customer data.</p>
          </div>
        </section>
      </main>
      <footer className="site-footer">
        <div>
          <Link to="/" className="brand-link">
            <Brand />
          </Link>
          <p>Synthetic investigation desk.</p>
        </div>
        <div className="footer-links">
          <a href="#platform">Platform</a>
          <a href="#workflow">Workflow</a>
          <Link to={entry}>Workspace</Link>
        </div>
        <div className="footer-bottom">
          <span>
            © {new Date().getFullYear()} FinGuard · Synthetic-data demonstrator
          </span>
        </div>
      </footer>
    </div>
  );
}
