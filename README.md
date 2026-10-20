# Caslingo

Caslingo is an open-source web platform for translating the user interface and the documentation of software projects.
It reads the source strings from your git repository, lets people translate them in the browser or fills them in with a
large language model, and gives the translations back: as commits or pull requests, or as a download at build time.

It is built with Go (Beego, XORM) and React (Vite, shadcn/ui), signs users in with [Casdoor](https://casdoor.ai), and runs
on SQLite or MySQL.

## Features

- **Projects and components**: a project is usually one repository; a component is a set of files of one format,
  like `web/src/locales/en/data.json` or `docs/**/*.{md,mdx}`, with a pattern for where each language's file goes.
- **Formats**:
  - JSON, nested or flat (i18next, Casdoor-style `data.json`); key order and indentation are kept.
  - Docusaurus JSON (`{"message", "description"}`, the files of `docusaurus write-translations`).
  - Markdown and MDX: paragraphs, headings, list items, table rows, quotes, note box titles and the front matter's
    `title`, `description` and `sidebar_label` are translated; code blocks, JSX, imports and comments stay as they are.
    Translated headings can keep the English heading's anchor (`{#anchor}`) so links to sections keep working.
- **Two-way git sync**: pulling imports the translations already in the repository, and later the ones changed there;
  when a string is changed both in the repository and in Caslingo, Caslingo's wins and is pushed next time. A changed
  source string marks its translations as needing an edit, and a Markdown paragraph with a few words changed keeps its
  old translation to start from.
- **Push** to the branch or as a GitHub pull request, with your own commit message; or turn pushing off and download the
  translations at build time.
- **Machine translation** with any OpenAI-compatible chat API (Qwen, DeepSeek, GPT, ...): the translation memory first
  (free), then the model in batches, following the project's glossary and instructions, keeping placeholders, inline
  code, tags and links. Results that break those are marked as needing an edit.
- **Editor**: filters (untranslated, needing an edit, failing checks, with suggestions, ...), search, per-file view,
  translation memory across projects, glossary matches, the other languages, history, Ctrl+Enter to save and go on.
- **Checks**: placeholders (`{name}`, `{{count}}`, `%s`), inline code, HTML/JSX tags, link targets and surrounding spaces.
- **People**: public or private projects, maintainers, translators, or open editing for anyone signed in; others make
  suggestions that translators accept.
- **History and contributors** for every project.
- **CI integration**: a GitHub webhook pulls on every push; an access token per project lets build jobs download a zip of
  the translations and upload source files that are generated at build time.

## Quick start

```bash
git clone https://github.com/caslingo/caslingo
cd caslingo/web && yarn install && yarn build && cd ..
go run main.go
```

Open http://localhost:16500. For local development without Casdoor, set `devSignin = true` in `conf/app.conf` (it only
accepts requests from 127.0.0.1) and sign in as `admin`. During frontend work, run `yarn dev` in `web/` and open
http://localhost:5173; it proxies `/api` to the backend.

With Docker:

```bash
docker build -t caslingo .
docker run -p 16500:16500 -v caslingo-data:/app/data caslingo
```

## Configuration

`conf/app.conf`; every item can also be set as an environment variable of the same name.

| Item | Meaning |
|---|---|
| `driverName`, `dataSourceName`, `dbName` | `sqlite` with `dbName = caslingo.db`, or `mysql` with `dataSourceName = user:password@tcp(host:3306)/` |
| `casdoorEndpoint`, `clientId`, `clientSecret`, `casdoorOrganization`, `casdoorApplication` | the Casdoor application users sign in with; its redirect URL is `https://<your site>/callback` |
| `admins` | Casdoor user names who can create projects; Casdoor's organization admins can too |
| `aiBaseUrl`, `aiModel`, `aiApiKey` | an OpenAI-compatible chat API, e.g. `https://dashscope.aliyuncs.com/compatible-mode/v1` with `qwen-flash` |
| `aiExtraBody` | JSON merged into each request, e.g. `{"enable_thinking": false}` for Qwen3 |
| `aiBatchSize`, `aiConcurrency` | strings per request, and requests at the same time |
| `reposDir`, `filesDir` | where repositories are cloned and uploaded files are kept |
| `syncIntervalMinutes` | pull every project this often (0: only on webhooks and the Pull button) |

## Using it from CI

Each project has an access token (project page, Integration tab).

Take the translations at build time, e.g. before `docusaurus build`:

```bash
curl -fsSL -H "Authorization: Bearer $CASLINGO_TOKEN" \
  "https://caslingo.example.com/api/download-translations?project=my-docs" -o translations.zip
unzip -o translations.zip && rm translations.zip
```

Upload source files generated at build time (for components marked "The source files are uploaded"):

```bash
curl -fsSL -H "Authorization: Bearer $CASLINGO_TOKEN" -H "Content-Type: application/octet-stream" \
  --data-binary @i18n/en/code.json "https://caslingo.example.com/api/upload-source?project=my-docs&path=i18n/en/code.json"
```

For the GitHub webhook, add `https://caslingo.example.com/api/webhook?project=<name>` for push events, content type
`application/json`, with the project's webhook secret.

## Translating Caslingo

Caslingo's own interface is written in English in the code as `t("namespace:Text")`. `yarn i18n` in `web/` collects
the texts into `web/src/locales/en/data.json`, and the other languages are translated in Caslingo.

## License

[Apache-2.0](LICENSE)
