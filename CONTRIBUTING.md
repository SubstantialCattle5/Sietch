<!-- omit in toc -->
# Contributing to Sietch Vault

First off, thanks for taking the time to contribute! ❤️

All types of contributions are encouraged and valued. See the [Table of Contents](#table-of-contents) for different ways to help and details about how this project handles them. Please make sure to read the relevant section before making your contribution. It will make it a lot easier for us maintainers and smooth out the experience for all involved. The community looks forward to your contributions. 🎉

> And if you like the project, but just don't have time to contribute, that's fine. There are other easy ways to support the project and show your appreciation, which we would also be very happy about:
>
> - Star the project
> - Tweet about it
> - Refer this project in your project's readme
> - Mention the project at local meetups and tell your friends/colleagues

<!-- omit in toc -->
## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [I Have a Question](#i-have-a-question)
- [I Want To Contribute](#i-want-to-contribute)
- [Reporting Bugs](#reporting-bugs)
- [Suggesting Enhancements](#suggesting-enhancements)
- [Your First Code Contribution](#your-first-code-contribution)
- [Improving The Documentation](#improving-the-documentation)
- [AI-Assisted Contributions](#ai-assisted-contributions)
- [Styleguides](#styleguides)
- [Commit Messages](#commit-messages)
- [Join The Project Team](#join-the-project-team)

## Code of Conduct

This project and everyone participating in it is governed by the
[Sietch Vault Code of Conduct](https://github.com/SubstantialCattle5/Sietch/blob/main/CODE_OF_CONDUCT.md).
By participating, you are expected to uphold this code. Please report unacceptable behavior
to [sietch@nilaysharan.com](mailto:sietch@nilaysharan.com).

## I Have a Question

> If you want to ask a question, we assume that you have read the available [Documentation](https://sietch.nilaysharan.com/).

Before you ask a question, it is best to search for existing [Issues](https://github.com/SubstantialCattle5/Sietch/issues) that might help you. In case you have found a suitable issue and still need clarification, you can write your question in this issue. It is also advisable to search the internet for answers first.

If you then still feel the need to ask a question and need clarification, we recommend the following:

- Open an [Issue](https://github.com/SubstantialCattle5/Sietch/issues/new).
- Provide as much context as you can about what you're running into.
- Provide project and platform versions (Go version, OS, etc), depending on what seems relevant.

We will then take care of the issue as soon as possible.

<!--
You might want to create a separate issue tag for questions and include it in this description. People should then tag their issues accordingly.

Depending on how large the project is, you may want to outsource the questioning, e.g. to Stack Overflow or Gitter. You may add additional contact and information possibilities:
- IRC
- Slack
- Gitter
- Stack Overflow tag
- Blog
- FAQ
- Roadmap
- E-Mail List
- Forum
-->

## I Want To Contribute

> ### Legal Notice <!-- omit in toc -->
>
> When contributing to this project, you must agree that you have authored 100% of the content, that you have the necessary rights to the content and that the content you contribute may be provided under the project licence.

### Reporting Bugs

<!-- omit in toc -->
#### Before Submitting a Bug Report

A good bug report shouldn't leave others needing to chase you up for more information. Therefore, we ask you to investigate carefully, collect information and describe the issue in detail in your report. Please complete the following steps in advance to help us fix any potential bug as fast as possible.

- Make sure that you are using the latest version.
- Determine if your bug is really a bug and not an error on your side e.g. using incompatible environment components/versions (Make sure that you have read the [documentation](https://sietch.nilaysharan.com/). If you are looking for support, you might want to check [this section](#i-have-a-question)).
- To see if other users have experienced (and potentially already solved) the same issue you are having, check if there is not already a bug report existing for your bug or error in the [bug tracker](https://github.com/SubstantialCattle5/Sietch/issues?q=label%3Abug).
- Also make sure to search the internet (including Stack Overflow) to see if users outside of the GitHub community have discussed the issue.
- Collect information about the bug:
- Stack trace (Traceback)
- OS, Platform and Version (Windows, Linux, macOS, x86, ARM)
- Version of the interpreter, compiler, SDK, runtime environment, package manager, depending on what seems relevant.
- Possibly your input and the output
- Can you reliably reproduce the issue? And can you also reproduce it with older versions?

<!-- omit in toc -->
#### How Do I Submit a Good Bug Report?

> You must never report security related issues, vulnerabilities or bugs including sensitive information to the issue tracker, or elsewhere in public. Instead sensitive bugs must be sent by email to [sietch@nilaysharan.com](mailto:sietch@nilaysharan.com).
<!-- You may add a PGP key to allow the messages to be sent encrypted as well. -->

We use GitHub issues to track bugs and errors. If you run into an issue with the project:

- Open an [Issue](https://github.com/SubstantialCattle5/Sietch/issues/new). (Since we can't be sure at this point whether it is a bug or not, we ask you not to talk about a bug yet and not to label the issue.)
- Explain the behavior you would expect and the actual behavior.
- Please provide as much context as possible and describe the *reproduction steps* that someone else can follow to recreate the issue on their own. This usually includes your code. For good bug reports you should isolate the problem and create a reduced test case.
- Provide the information you collected in the previous section.

Once it's filed:

- The project team will label the issue accordingly.
- A team member will try to reproduce the issue with your provided steps. If there are no reproduction steps or no obvious way to reproduce the issue, the team will ask you for those steps and mark the issue as `needs-repro`. Bugs with the `needs-repro` tag will not be addressed until they are reproduced.
- If the team is able to reproduce the issue, it will be marked `needs-fix`, as well as possibly other tags (such as `critical`), and the issue will be left to be [implemented by someone](#your-first-code-contribution).

<!-- You might want to create an issue template for bugs and errors that can be used as a guide and that defines the structure of the information to be included. If you do so, reference it here in the description. -->

### Suggesting Enhancements

This section guides you through submitting an enhancement suggestion for Sietch Vault, **including completely new features and minor improvements to existing functionality**. Following these guidelines will help maintainers and the community to understand your suggestion and find related suggestions.

<!-- omit in toc -->
#### Before Submitting an Enhancement

- Make sure that you are using the latest version.
- Read the [documentation](https://sietch.nilaysharan.com/) carefully and find out if the functionality is already covered, maybe by an individual configuration.
- Perform a [search](https://github.com/SubstantialCattle5/Sietch/issues) to see if the enhancement has already been suggested. If it has, add a comment to the existing issue instead of opening a new one.
- Find out whether your idea fits with the scope and aims of the project. It's up to you to make a strong case to convince the project's developers of the merits of this feature. Keep in mind that we want features that will be useful to the majority of our users and not just a small subset. If you're just targeting a minority of users, consider writing an add-on/plugin library.

<!-- omit in toc -->
#### How Do I Submit a Good Enhancement Suggestion?

Enhancement suggestions are tracked as [GitHub issues](https://github.com/SubstantialCattle5/Sietch/issues).

- Use a **clear and descriptive title** for the issue to identify the suggestion.
- Provide a **step-by-step description of the suggested enhancement** in as many details as possible.
- **Describe the current behavior** and **explain which behavior you expected to see instead** and why. At this point you can also tell which alternatives do not work for you.
- You may want to **include screenshots or screen recordings** which help you demonstrate the steps or point out the part which the suggestion is related to. You can use [LICEcap](https://www.cockos.com/licecap/) to record GIFs on macOS and Windows, and the built-in [screen recorder in GNOME](https://help.gnome.org/users/gnome-help/stable/screen-shot-record.html.en) or [SimpleScreenRecorder](https://github.com/MaartenBaert/ssr) on Linux. <!-- this should only be included if the project has a GUI -->
- **Explain why this enhancement would be useful** to most Sietch Vault users. You may also want to point out the other projects that solved it better and which could serve as inspiration.

<!-- You might want to create an issue template for enhancement suggestions that can be used as a guide and that defines the structure of the information to be included. If you do so, reference it here in the description. -->

### Your First Code Contribution

Welcome to your first code contribution! Here's how to get started:

#### Development Environment Setup

1. **Prerequisites**:
   - Go 1.19 or later
   - Git
   - Make (for build automation)

2. **Fork and Clone**:

   ```bash
   git clone https://github.com/YOUR_USERNAME/Sietch.git
   cd Sietch
   ```

3. **Build the Project**:

   ```bash
   make build
   ```

4. **Run Tests**:

   ```bash
   make test
   ```

5. **Install Development Tools**:

   ```bash
   make setup-hooks  # Sets up git hooks
   ```

#### Making Your First Contribution

1. Look for issues labeled `good first issue` or `help wanted`
2. Comment on the issue to let others know you're working on it
3. Create a new branch: `git checkout -b feature/your-feature-name`
4. Make your changes following our [style guidelines](#styleguides)
5. Write or update tests as needed
6. Run `make test` to ensure all tests pass
7. Commit your changes with a descriptive commit message
8. Push to your fork and create a pull request

#### Code Structure

- `cmd/` - CLI command implementations
- `internal/` - Internal packages (not for external use)
- `util/` - Utility functions
- `testutil/` - Testing utilities
- `template/` - Configuration templates

### Improving The Documentation

Documentation improvements are always welcome! Here are ways you can help:

#### Types of Documentation

1. **Code Documentation**:
   - Add or improve Go doc comments for functions, types, and packages
   - Follow Go documentation conventions
   - Include examples where helpful

2. **User Documentation**:
   - Update the main README.md
   - Improve command help text
   - Add usage examples
   - Update configuration templates

3. **Developer Documentation**:
   - Architecture decisions
   - API documentation
   - Contributing guidelines (this file!)

#### Documentation Standards

- Use clear, concise language
- Include practical examples
- Keep documentation up-to-date with code changes
- Test any code examples to ensure they work
- Use proper Markdown formatting

#### Submitting Documentation Changes

1. Follow the same process as code contributions
2. For small fixes, you can edit directly on GitHub
3. For larger changes, clone the repository and test locally
4. Ensure all links work and formatting is correct

## AI-Assisted Contributions

We recognize that AI tools (GitHub Copilot, ChatGPT, Claude, etc.) are increasingly used in software development. We welcome AI-assisted contributions, but with clear expectations around quality and disclosure.

### Our AI Policy

#### 1. **Disclosure is Required**
- You **must** disclose if you used AI tools to generate or significantly modify code
- Check the "AI assistance" box in the PR template
- Briefly describe which parts were AI-assisted and how you validated them

#### 2. **You Are Responsible**
- AI-generated code must be reviewed, tested, and understood by you
- You are accountable for all code you submit, regardless of origin
- Ensure the code follows our style guidelines and best practices
- Don't submit code you don't understand

#### 3. **Quality Over Quantity**
- AI-assisted PRs are judged by the same quality standards as any other PR
- Bulk AI-generated changes without clear value will be rejected
- Focus on solving real problems, not just generating code

#### 4. **What We Don't Accept**
- ❌ Copy-pasted AI output without review or testing
- ❌ Bulk formatting/refactoring PRs with no functional improvement
- ❌ Generic documentation changes ("improved clarity") without substance
- ❌ PRs that don't link to an existing issue or explain the problem being solved
- ❌ Multiple low-effort PRs submitted rapidly (spam pattern)

#### 5. **What We Do Accept**
- ✅ Well-tested code that happens to be AI-assisted
- ✅ AI-generated boilerplate that you've reviewed and customized
- ✅ Documentation improvements that add real clarity
- ✅ Bug fixes where AI helped you understand the issue
- ✅ Test cases generated by AI that you've validated

### Detection and Labeling

We use automated workflows to detect likely AI-generated content:
- PRs may be automatically labeled `ai-assisted` or `ai-suspected`
- This is not a rejection - just a signal for extra human review
- If you disclosed AI usage, you'll get the `ai-assisted` label proactively

### Hacktoberfest and Spam Prevention

During Hacktoberfest (and year-round), we actively monitor for low-effort contributions:
- PRs without linked issues will be flagged
- Trivial changes (typo fixes, whitespace, lockfile-only) need strong justification
- Generic or template-style PR descriptions will trigger review
- Multiple rapid PRs from one author will be examined for spam patterns

If your PR is labeled `needs-justification`:
1. Link to a relevant issue (create one if needed)
2. Explain the value your change provides
3. Fill out the PR template completely
4. Respond to reviewer feedback

**PRs that don't address justification requests within 48 hours may be closed.**

### Best Practices for AI-Assisted Contributions

1. **Start with a real issue**: Identify a genuine problem or enhancement need
2. **Use AI as a tool, not a shortcut**: Let AI help with boilerplate, but add your expertise
3. **Review thoroughly**: Understand every line of AI-generated code
4. **Test comprehensively**: Write tests or verify existing tests cover your changes
5. **Document your process**: In the PR, explain what you did and why
6. **Engage with reviewers**: Be responsive to feedback and willing to iterate

### Why This Policy Exists

We want to:
- Maintain high code quality and project integrity
- Prevent spam and low-effort contributions that waste maintainer time
- Ensure contributors learn and grow, not just generate code
- Build a sustainable, community-driven project

AI is a powerful tool when used responsibly. We encourage thoughtful, well-tested contributions regardless of the tools you use to create them.

## Styleguides

### Go Code Style

- Follow standard Go formatting (`go fmt`)
- Use `golint` and `go vet` to check your code
- Write clear, descriptive variable and function names
- Add comments for exported functions and types
- Keep functions focused and reasonably sized
- Handle errors appropriately - don't ignore them
- Use meaningful package names

### Commit Messages

We follow the [Conventional Commits](https://www.conventionalcommits.org/) specification:

```text
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

#### Types

- `feat`: A new feature
- `fix`: A bug fix
- `docs`: Documentation only changes
- `style`: Changes that do not affect the meaning of the code
- `refactor`: A code change that neither fixes a bug nor adds a feature
- `perf`: A code change that improves performance
- `test`: Adding missing tests or correcting existing tests
- `chore`: Changes to the build process or auxiliary tools

#### Examples

```text
feat(encryption): add AES-256 encryption support

fix(vault): resolve issue with vault initialization on Windows

docs: update installation instructions for macOS

test(dedup): add unit tests for deduplication manager
```

#### Guidelines

- Use the imperative mood ("add feature" not "added feature")
- Keep the first line under 72 characters
- Reference issues and pull requests when relevant
- Explain the "why" in the body, not just the "what"

## Join The Project Team

Interested in becoming a more involved contributor? We'd love to have you join the team!

### Ways to Get More Involved

1. **Regular Contributor**:
   - Consistently contribute high-quality code or documentation
   - Help review pull requests from other contributors
   - Participate in discussions about project direction

2. **Maintainer**:
   - Help triage issues and manage the project roadmap
   - Review and merge pull requests
   - Make decisions about project architecture and features

3. **Core Team Member**:
   - Shape the long-term vision of the project
   - Represent the project in the community
   - Mentor new contributors

### How to Apply

If you're interested in taking on a larger role:

1. Demonstrate consistent, quality contributions over time
2. Show good judgment in code reviews and discussions
3. Help other contributors and be welcoming to newcomers
4. Reach out to current maintainers at [sietch@nilaysharan.com](mailto:sietch@nilaysharan.com)

### Responsibilities

With greater involvement comes greater responsibility:

- **Code Quality**: Maintain high standards for code and documentation
- **Community**: Foster a welcoming, inclusive environment
- **Communication**: Respond to issues and pull requests in a timely manner
- **Vision**: Help guide the project's technical and strategic direction
- **Mentorship**: Help onboard new contributors

### Recognition

We believe in recognizing our contributors:

- Contributors are listed in our README
- Significant contributors may be invited to join the GitHub organization
- We highlight contributions in release notes
- Annual contributor appreciation posts

<!-- omit in toc -->
## Attribution

This guide is based on the [contributing.md](https://contributing.md/generator)!
