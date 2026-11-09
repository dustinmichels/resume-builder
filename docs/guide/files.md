# Your files & versions

Everything lives in plain files in your data folder, so you can back it up, sync it with Dropbox or iCloud, or put it in
git like any other folder. The folder button in the toolbar opens the current resume's folder.

```
~/resume-builder/
  Resume/
    resume.json                               the resume, autosaved as you edit
    resume.md, .docx, .pdf                    copies written by Sync files
    additional/                               details for applications, never printed
      contact.json                            Contact tab
      supervisors.json                        Supervisors tab
    cover-letters/
      Metro Network Planner.json              a cover letter
      Metro Network Planner.md, .docx, .pdf   its copies, written by Sync files
    versions/
      20261106-153000 Sent to Acme/           a saved version: timestamp, then its label
        resume.json, additional/, cover-letters/
        resume.md, .docx, .pdf
  Designer CV/
    resume.json
  .trash/
    Old draft 20261101-090000/                deleted resumes, moved here rather than erased
```

Each folder holding a `resume.json` is a resume, and its folder name is the name shown in the editor. Renaming a resume
renames its folder.

## Versions

A version is a named snapshot, for example "Sent to Acme". The **Versions** tab saves one from the resume, its cover
letters and its contact and supervisor details as last saved, and writes Markdown, Word and PDF copies of the resume
and each letter inside it. Six months later you can open exactly what you sent.

| Action                  | What happens                                                                                       |
| ----------------------- | -------------------------------------------------------------------------------------------------- |
| **Restore**             | Replaces the resume with the version. The current state is saved first as `Before restoring …`, so a restore can always be undone |
| **Copy to new resume**  | Starts a new resume from the version, letters and details included                                 |
| ×                       | Deletes the version and its copies                                                                 |

Restoring replaces only the resume itself; your letters and contact details stay as they are.

## Contact and supervisors

The **Contact** and **Supervisors** tabs hold details job applications ask for that don't belong on the page: your
mailing address and phone, and for each job a supervisor's name, position, contact details and whether they may be
contacted. They autosave with the resume and are never rendered or exported.

Supervisors are matched to entries. If you delete an entry, its supervisor is listed under **No longer in the resume**
until you remove it.

## Deleting

Deleting a resume moves its folder, versions included, into `.trash`. Deleting a cover letter moves the letter alone.
Empty `.trash` by hand when you no longer need what's in it.
