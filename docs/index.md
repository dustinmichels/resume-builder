---
layout: home

hero:
  name: Resume Builder
  text: Write it once. Read right everywhere.
  tagline: A local resume editor. Edit on the page, see the exact PDF as you type, and check what applicant tracking systems and screen readers will read.
  image:
    src: /app-icon.svg
    alt: Resume Builder app icon
  actions:
    - theme: brand
      text: Download for Mac
      link: https://github.com/dustinmichels/resume-builder/releases/latest/download/Resume-Builder.dmg
    - theme: alt
      text: Get started
      link: /guide/install
    - theme: alt
      text: View on GitHub
      link: https://github.com/dustinmichels/resume-builder
---

<div class="home-body">

<img class="shot hero-shot" src="/screenshots/editor.webp" alt="The Resume Builder editor: content forms on the left, the live PDF page on the right" />

<FeatureGrid />

<Showcase />

<div class="get-it">
  <h2>Get it</h2>
  <p>Free and open source. Runs on macOS, Windows and Linux; your resumes never leave your computer.</p>
  <div class="ways">
    <a href="./guide/install#mac">
      <strong>Mac app</strong>
      <span>Signed and notarized. Drag to Applications, updates itself.</span>
    </a>
    <a href="./guide/install#command-line">
      <strong>Command line</strong>
      <span>macOS, Windows and Linux. Opens the editor in your browser.</span>
    </a>
    <a href="./developers/">
      <strong>From source</strong>
      <span>Go, Vue and Typst. One <code>mise run dev</code> away.</span>
    </a>
  </div>
</div>

</div>

<style>
.home-body {
  max-width: 1152px;
  margin: 0 auto;
  padding: 0 24px 96px;
}

@media (min-width: 640px) {
  .home-body {
    padding: 0 48px 96px;
  }
}

.hero-shot {
  margin: 8px 0 72px;
}

.get-it {
  margin-top: 96px;
  text-align: center;
}

.home-body .get-it h2 {
  margin: 0 0 8px;
  padding-top: 0;
  border-top: none;
  font-size: 28px;
  font-weight: 700;
}

.get-it > p {
  margin: 0 0 28px;
  color: var(--vp-c-text-2);
}

.ways {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
  gap: 16px;
  text-align: left;
}

.home-body .ways a {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 20px 24px;
  border: 1px solid var(--vp-c-divider);
  border-radius: 12px;
  color: inherit;
  text-decoration: none;
  transition: border-color 0.25s;
}

.home-body .ways a:hover {
  border-color: var(--vp-c-brand-1);
}

.ways strong {
  color: var(--vp-c-brand-1);
}

.ways span {
  font-size: 14px;
  color: var(--vp-c-text-2);
}
</style>
