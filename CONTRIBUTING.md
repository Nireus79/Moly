# Contributing to Moly

Thank you for your interest in contributing to Moly! We welcome contributions from the community.

## Code of Conduct

- Be respectful and inclusive
- Focus on the work, not the person
- Help others learn and grow
- Report issues constructively

## How to Contribute

### Reporting Issues

1. Check if the issue already exists
2. Provide a clear description
3. Include steps to reproduce
4. Share your environment (OS, Chrome version, etc.)

### Submitting Pull Requests

1. **Fork the repository**
   ```bash
   git clone https://github.com/YOUR_USERNAME/Moly.git
   cd Moly/moly-extension
   ```

2. **Create a feature branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```

3. **Make your changes**
   - Follow existing code style
   - Write clear commit messages
   - Keep commits focused and atomic

4. **Test your changes**
   ```bash
   npm run build
   npm run lint
   npm run test
   ```

5. **Push and create PR**
   ```bash
   git push origin feature/your-feature-name
   ```

6. **Describe your changes**
   - What problem does this solve?
   - How did you test it?
   - Any breaking changes?

## Development Setup

### Prerequisites

- Node.js 18+
- npm or yarn
- Chrome/Chromium browser

### Getting Started

```bash
# Install dependencies
cd moly-extension
npm install

# Development build
npm run dev

# Production build
npm run build

# Linting
npm run lint

# Type checking
npm run type-check
```

### Loading the Extension

1. Open `chrome://extensions/`
2. Enable "Developer mode"
3. Click "Load unpacked"
4. Select `moly-extension/dist` folder

## Architecture

- **Extension**: `moly-extension/` - React + TypeScript UI
- **Native Host**: `moly-installer/native-host/` - Python service for system control
- **Installer**: `moly-installer/` - Setup orchestration

See `INSTALLATION_ARCHITECTURE.md` for details.

## Areas for Contribution

### High Priority

- macOS and Windows native host binaries
- Cloud sync backend
- Team collaboration features
- Advanced AI coaching algorithms

### Medium Priority

- UI/UX improvements
- Performance optimization
- Localization (i18n)
- Documentation

### Community Ideas

Have an idea? Open an issue to discuss before starting work.

## Code Standards

- **Language**: TypeScript (strict mode)
- **Formatting**: Prettier (auto-format on save)
- **Linting**: ESLint (no warnings)
- **Style**: Functional components, React hooks, Zustand for state
- **Testing**: Jest + React Testing Library

## Commit Messages

Use clear, descriptive messages:

```
feat: Add cloud sync for premium users
fix: Resolve native host timeout on slow connections
docs: Update installation guide for macOS
refactor: Simplify model detection logic
test: Add E2E tests for uninstall flow
```

## Pull Request Process

1. One feature per PR
2. Include tests for new features
3. Update documentation as needed
4. Respond to review feedback
5. Keep commits clean (rebase if needed)

## Getting Help

- Check `QUICKSTART.md` for setup help
- See `TROUBLESHOOTING.md` for common issues
- Open a discussion for architecture questions
- Ask in issues before diving into complex work

## Recognition

Contributors are recognized in:
- Release notes
- Hall of fame (coming soon)
- Sponsor tier (coming soon)

## License

By contributing, you agree that your contributions will be licensed under the MIT License.

---

Thank you for making Moly better! 🙏

Questions? Open an issue or start a discussion.
