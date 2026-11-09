# Editing

## Click the page to edit

Click any text on the **Page** preview to edit that value in place. A small editor opens next to it, and the page
re-renders as you type.

![Clicking a bullet on the page opens an editor beside it, and the sidebar scrolls to the same field](/screenshots/click-to-edit.webp)

| Key                                 | In the page editor                                                |
| ----------------------------------- | ----------------------------------------------------------------- |
| <kbd>Enter</kbd>                    | Add the next bullet, paragraph, column line or headline part      |
| <kbd>Backspace</kbd> in an empty one | Remove it                                                        |
| <kbd>Tab</kbd> / <kbd>⇧Tab</kbd>    | Move to the next / previous value                                 |
| <kbd>Esc</kbd>                      | Close the editor                                                  |

The sidebar follows along: it switches to **Content** (or **Cover letters**, for a letter), opens the section and entry
holding the value, closes the others, scrolls to the field and briefly highlights it.

## Formatting

Select text and press:

| Keys                          | Format                |
| ----------------------------- | --------------------- |
| <kbd>⌘B</kbd>                 | **bold**              |
| <kbd>⌘I</kbd>                 | *italic*              |
| <kbd>⌘K</kbd>                 | link                  |
| <kbd>⇧Enter</kbd>             | line break            |

On Windows and Linux, use <kbd>Ctrl</kbd> for <kbd>⌘</kbd>. Formatting works in paragraphs, subtitles, bullets,
subheadings and column lines, and in a letter's paragraphs, bullets and header lines. Other fields (names, dates,
locations) are plain text.

Characters like `#`, `$` and `@` are printed as typed; there's nothing to escape.

## Icons beside lines

Column lines can carry an icon, hung in the margin to the left of the line. Click the image button beside a line in the
sidebar, an icon on the page, or the image button in the page editor, then search
[Lucide's icon set](https://lucide.dev/icons) (<kbd>Enter</kbd> picks the first match). A template with its own icons,
like [Julia](./templates#julia), lists those first.

Icons are decorations: they never appear in the PDF's text layer, the screen-reader tags or the Markdown export, and
they're marked decorative in Word.

## Columns side by side

In a side-by-side columns section, the dashed line between two columns on the page moves where the next column starts.
Drag it, or focus it and use the arrow keys (<kbd>Shift</kbd> for bigger steps). Double-click the line to reset the
columns to fit their content.

## Reorder and hide

- Drag the grip (⋮⋮) beside a section, entry, bullet or line to move it.
- The A→Z button on a line sorts the comma-separated items within it, such as `Python, SQL, GIS`.
- The eye button hides a section, entry, subheading or column from the page and every export, without deleting it.
  Click it again to bring it back.

## Undo

<kbd>⌘Z</kbd> undoes resume edits, including inside text fields; <kbd>⇧⌘Z</kbd> redoes (<kbd>Ctrl</kbd>+<kbd>Z</kbd> and
<kbd>Ctrl</kbd>+<kbd>Y</kbd> elsewhere). The arrows next to the save status do the same. Typing with pauses under a
second counts as one step.

History covers the Content and Style tabs and the page editor, and resets when you open another resume or restore a
version. On the **Cover letters** tab the same keys act on the open letter, which has its own history.
