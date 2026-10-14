# Contributing to LexScale (律衡)

Thank you for your interest in contributing to LexScale!

## Code of Conduct

Be respectful, inclusive, and professional.

## How to Contribute

### 1. Fork & Clone

```bash
git clone https://github.com/your-username/lexscale.git
cd lexscale
```

### 2. Set Up Development Environment

```bash
# Install Go 1.23+
go mod download

# Run tests
go test ./...

# Build
go build ./...
```

### 3. Make Changes

- Write clean, idiomatic Go code
- Add tests for new features
- Update documentation
- Follow existing code style

### 4. Submit Pull Request

- Create a feature branch: `git checkout -b feat/your-feature`
- Commit with clear messages
- Push and create PR

## Development Guidelines

### Code Style

- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `gofmt` and `go vet`
- Write meaningful comments
- Keep functions small and focused

### Testing

- Write unit tests for all new code
- Aim for >80% coverage
- Use table-driven tests
- Mock external dependencies

### Commit Messages

```
feat: add new contract analysis rule
fix: correct risk level calculation
docs: update API documentation
test: add tests for guard gate
refactor: simplify engine logic
```

## Areas Needing Contributions

| Area | Description | Difficulty |
|------|-------------|------------|
| **Legal Rules** | Add more contract analysis rules | Easy |
| **Chinese Law DB** | Integrate legal databases | Medium |
| **Python SDK** | Build Python client library | Medium |
| **Frontend** | Improve React dashboard | Medium |
| **Documentation** | Write tutorials/examples | Easy |
| **Testing** | Increase test coverage | Easy |

## Questions?

Open an issue or discussion. We're here to help!

---

**LexScale — Where Law Meets Intelligence**
