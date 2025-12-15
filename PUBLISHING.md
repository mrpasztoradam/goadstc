# Publishing to pkg.go.dev

## Prerequisites ✅
- [x] GitHub repository exists: https://github.com/mrpasztoradam/goadstc.git
- [x] go.mod file is present with correct module path
- [x] Library version system implemented (v0.1.0)
- [ ] All changes committed and pushed
- [ ] Git tag created
- [ ] Tag pushed to GitHub

## Publishing Steps

### 1. Commit Your Current Changes
```bash
# Stage modified files
git add AI_AGENT_GUIDE.md CHANGELOG.md version.go version_test.go

# Commit
git commit -m "chore: prepare v0.1.0 release

Update version information and documentation for initial release."
```

### 2. Push Your Feature Branch
```bash
git push origin feat/connection-stability-performance
```

### 3. (Optional but Recommended) Merge to Main
If you want to tag on main branch:
```bash
git checkout main
git pull origin main
git merge feat/connection-stability-performance
git push origin main
```

Or tag on the feature branch (works too).

### 4. Create and Push the Version Tag
```bash
# Create annotated tag (recommended for releases)
git tag -a v0.1.0 -m "Release v0.1.0

Initial release with:
- Full ADS/AMS protocol implementation
- 34 type-safe read/write methods
- Symbol resolution and notifications
- Connection stability (auto-reconnect, health checks)
- Observability (logging, metrics, error classification)
- Comprehensive documentation and examples"

# Push the tag to GitHub
git push origin v0.1.0
```

### 5. Trigger pkg.go.dev Indexing

Option A - Automatic (wait 15-30 minutes):
- pkg.go.dev automatically discovers new modules from GitHub

Option B - Manual (instant):
- Visit: https://pkg.go.dev/github.com/mrpasztoradam/goadstc@v0.1.0
- This will trigger immediate indexing

### 6. Verify Publication
After a few minutes, your package will appear at:
- Main package: https://pkg.go.dev/github.com/mrpasztoradam/goadstc
- Specific version: https://pkg.go.dev/github.com/mrpasztoradam/goadstc@v0.1.0

## What Gets Published

pkg.go.dev will automatically:
- ✅ Display your README.md as the package overview
- ✅ Show all exported functions, types, and their documentation
- ✅ Generate documentation from your code comments
- ✅ List available versions and tags
- ✅ Show examples from your examples/ directory
- ✅ Display the module's dependencies (none in your case!)
- ✅ Show the LICENSE file
- ✅ Provide import instructions

## Best Practices

### Documentation Tips
Your package is well-documented! But ensure:
- All exported types have doc comments starting with the type name
- Package comment in client.go explains the library purpose
- Examples use standard Go example format

### Version Tags
- Use semantic versioning: v0.1.0, v0.2.0, v1.0.0
- Always prefix with 'v': v0.1.0 (not 0.1.0)
- Use annotated tags (git tag -a) for releases
- Breaking changes require major version bump (v2.0.0)

### Module Path
Your module path is correct: `github.com/mrpasztoradam/goadstc`
Users will import it as:
```go
import "github.com/mrpasztoradam/goadstc"
```

## After Publishing

### Update README.md Badge
Add a pkg.go.dev badge to your README:
```markdown
[![Go Reference](https://pkg.go.dev/badge/github.com/mrpasztoradam/goadstc.svg)](https://pkg.go.dev/github.com/mrpasztoradam/goadstc)
```

### Announce
- Update GitHub repository description
- Add topics/tags to GitHub repo (go, ads, twincat, beckhoff, automation, plc)
- Consider posting to r/golang or Go community forums

## Troubleshooting

**Package doesn't appear?**
- Ensure tag is pushed: `git ls-remote --tags origin`
- Check module path matches repository: look at go.mod
- Visit the URL directly to trigger indexing
- Wait up to 30 minutes for automatic discovery

**"Module not found" error?**
- Verify go.mod has correct module path
- Ensure repository is public
- Check that go.mod is in the repository root

**Documentation not showing?**
- Ensure comments start with the symbol name
- Use standard Go doc format (no markdown in doc comments)
- Package comment should be before "package goadstc"

## Current Status

You have 3 unpushed commits on your feature branch:
1. feat: add structured logging, error classification, and metrics
2. feat: add library versioning system  
3. docs: add AI agent development guide

Plus 4 modified files ready to commit.

Ready to proceed? I can help you with any of these steps!
