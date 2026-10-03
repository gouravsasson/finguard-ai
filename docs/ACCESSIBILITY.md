# Accessibility

The desk uses a skip link, labeled form controls, button names, `role="alert"` for errors, and `role="status"` for loading. Focus is visible. The graph has a text alternative in the limitations list under the picture. Layout stacks below 900px. Color is not the only status signal; status and triage words are printed.

This is a manual checklist for the demo, not a formal WCAG audit.

## Automated browser checks (2026-10-04)

Playwright and axe scan the landing page, sign-in, queue, and case workspace against WCAG 2 A/AA and WCAG 2.1 AA rules. Tested secondary text contrast passes. Evidence and graph scroll regions are keyboard focusable. Desktop, tablet, and mobile layouts are checked for document overflow. These checks cover the exercised screens and states; they are not a formal accessibility certification or a substitute for assistive-technology user testing.
