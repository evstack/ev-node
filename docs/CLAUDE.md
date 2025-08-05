# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Common Development Commands

### Local Development

- `yarn install` - Install dependencies
- `yarn dev` - Start VitePress development server (hot reload enabled)
- `yarn build` - Build static documentation site
- `yarn preview` - Preview built site locally
- `make testlink` - Validate all links in markdown files using lychee

### Deployment

- Automatic deployment via GitHub Actions on push to main branch
- Preview deployments available for pull requests

## High-Level Architecture

This is a **VitePress-based documentation site** for Evolve. Key architectural elements:

### Content Organization

```
docs/
├── .vitepress/config.ts    # Site configuration, navigation, theme
├── blog/                   # Blog posts and announcements
├── guides/                 # Step-by-step tutorials
│   ├── da/                # Data availability guides
│   ├── deploy/            # Deployment guides
│   ├── evm/               # EVM integration guides
│   └── execution/         # Execution layer guides
├── learn/                  # Technical documentation
│   ├── sequencing/        # Sequencing concepts
│   └── specs/             # Technical specifications
└── public/                 # Static assets and installation scripts
```

### Key Technical Details

- **Static Site Generator**: VitePress with Mermaid diagram support
- **Navigation Structure**: Configured in `.vitepress/config.ts` with collapsible sidebar sections
- **Analytics**: Plausible analytics and Chatbase integration
- **Content Format**: Markdown files with frontmatter support
- **Link Validation**: Automated via `make testlink` and GitHub Actions

### Development Workflow

1. Content changes: Edit markdown files in appropriate directories
2. Navigation changes: Update `.vitepress/config.ts`
3. Test locally with `yarn dev`
4. Validate links with `make testlink`
5. Create PR - preview deployment will be available
6. Merge to main triggers automatic deployment

### Important Conventions

- Use relative links for internal documentation references
- Place images in `/public/img/`
- Blog posts should include proper frontmatter with date and author
- Guides should be structured with clear step-by-step instructions
- Technical documentation in `/learn` should be comprehensive and accurate
