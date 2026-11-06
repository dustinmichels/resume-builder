// Fonts installed on macOS by default, so Typst can always find them, plus
// Libertinus Serif, which Typst bundles. Hoefler Text is left out: its narrow
// word spaces make pdftotext run words together.
export const FONTS = [
  'Helvetica',
  'Helvetica Neue',
  'Arial',
  'Avenir Next',
  'Gill Sans',
  'Optima',
  'Georgia',
  'Palatino',
  'Baskerville',
  'Charter',
  'Athelas',
  'Iowan Old Style',
  'Libertinus Serif',
]

/** The fonts to offer: a document saved with a font outside the list keeps it as the first option. */
export const fontOptions = (current: string) => (FONTS.includes(current) ? FONTS : [current, ...FONTS])
