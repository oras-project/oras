# ORAS Governance

The following document outlines ORAS project governance.

## The ORAS Project

The ORAS project consists of several repositories known as subprojects that enable community cohorts to experiment and implement solutions across the scope of the project.

## Maintainers Structure

There are two types of maintainers in the ORAS project organized hierarchically. ORAS org maintainers oversee the overall project and its health. Subproject maintainers focus on a single codebase, a group of related codebases, a service (e.g., a website), or subproject to support the other subprojects (e.g., marketing or community management). 

Changes in maintainership have to be announced via an [issue][oras-issues-new].

### Maintainer Responsibility

ORAS maintainers adhere to the requirements and responsibilities set forth in the respective [ORAS Org Maintainers](#oras-org-maintainers) and [Subproject Maintainers](#subproject-maintainers). They further pledge the following:

- To act in the best interest of the project and subprojects at all times.
- To ensure that project and subproject development and direction is a function of community needs.
- To never take any action while hesitant that it is the right action to take.
- To fulfill the responsibilities outlined in this document and its dependents.

### ORAS Org Maintainers

The [ORAS Org maintainers](OWNERS) are responsible for:

- Maintaining the mission, vision, values, and scope of the project
- Refining the governance and charter as needed
- Making project level decisions
- Resolving escalated project decisions when the subproject maintainers responsible are blocked
- Managing the ORAS brand
- Controlling access to ORAS assets such as source repositories, hosting, project calendars
- Deciding what subprojects are part of the ORAS project
- Deciding on the creation of new subprojects
- Overseeing the resolution and disclosure of security issues
- Managing financial decisions related to the project

Changes to org maintainers use the following:

- Any subproject maintainer is eligible for a position as an org maintainer
- No one company or organization can employ a simple majority of the org maintainers
- An org maintainer may step down by submitting an [issue][oras-issues-new] stating their intent and they will be moved to emeritus.
- Org maintainers MUST remain active on the project. If they are unresponsive for > 6 months they will lose org maintainership unless a [super-majority][super-majority] of the other org maintainers agrees to extend the period to be greater than 6 months
- When there is an opening for a new org maintainer, any current maintainer may nominate a suitable subproject maintainer as a replacement
  - Nominations for new maintainers must be made by creating an [issue][oras-issues-new].
- When nominated individual(s) agrees to be a candidate for maintainership, the subproject maintainers may vote
  - The voting period will be open for a minimum of three business days and will remain open until a super-majority of project maintainers has voted
  - Only current org maintainers are eligible to vote via casting a single vote each via a +1 comment on the nomination issue
  - Once a [super-majority][super-majority] has been reached the maintainer elect must complete [onboarding](#onboarding-a-new-maintainer) prior to becoming an official ORAS maintainer.
  - Once the maintainer onboarding has been completed a pull request is made on the repo adding the new maintainer to the [OWNERS](OWNERS) file.
- When an org maintainer steps down, they become an emeritus maintainer

Once an org maintainer is elected, they remain a maintainer until stepping down (or, in rare cases, are removed). Voting for new maintainers occurs when necessary to fill vacancies. Any existing subproject maintainer is eligible to become an org maintainer.

The Org Maintainers will select a chair to set agendas and call meetings of the Org Maintainers. Chairs will serve a term of 6 months, once the term is complete org maintainers will select a subsequent chair.

### Subproject Maintainers

Subproject maintainers are responsible for activities surrounding the development and release of content (eg. code, specifications, documentation) or the tasks needed to execute their subproject (e.g., community management) within the designated repository, or repositories associated with the subproject (e.g., community management). Technical decisions for code resides with the subproject maintainers unless there is a decision related to cross maintainer groups that cannot be resolved by those groups. Those cases can be escalated to the org maintainers.

Subproject maintainers many be responsible for one or many repositories.

Subproject maintainers do not need to be software developers. No explicit role is placed upon them and they can be anyone appropriate for the work being produced. For example, if a repository is for documentation it would be appropriate for maintainers to be technical writers.

Changes to maintainers use the following:

- A subproject maintainer may step down by submitting an [issue][oras-issues-new] stating their intent and they will be moved to emeritus.
- Maintainers MUST remain active. If they are unresponsive for > 6 months they will be automatically removed unless a [super-majority][super-majority] of the other subproject maintainers agrees to extend the period to be greater than 6 months
- New maintainers can be added to a subproject by a [super-majority][super-majority] vote of the existing maintainers
- When a subproject has no maintainers the ORAS org maintainers become responsible for it and may archive the subproject or find new maintainers

### Onboarding a New Maintainer

New ORAS maintainers participate in an onboarding period during which they fulfill all code review and issue management responsibilities that are required for their role. The length of this onboarding period is variable, and is considered complete once both the existing maintainers and the candidate maintainer are comfortable with the candidate's competency in the responsibilities of maintainership. This process MUST be completed prior to the candidate being named an official ORAS maintainer.

The onboarding period is intended to ensure that the to-be-appointed maintainer is able/willing to take on the time requirements, familiar with ORAS core logic and concepts, understands the overall system architecture and interactions that comprise it, and is able to work well with both the existing maintainers and the community.

## Decision Making at the ORAS org level

When maintainers need to make decisions there are two ways decisions are made, unless described elsewhere.

The default decision making process is [lazy-consensus][lazy-consensus]. This means that any decision is considered supported by the team making it as long as no one objects. Silence on any consensus decision is implicit agreement and equivalent to explicit agreement. Explicit agreement may be stated at will.

When a consensus cannot be found a maintainer can call for a [majority][majority] vote on a decision.

Many of the day-to-day project maintenance can be done by a lazy consensus model. But the following items must be called to vote:

- Removing a maintainer for any reason other than inactivity (super majority)
- Changing the governance rules (this document) (super majority)
- Licensing and intellectual property changes (including new logos, wordmarks) (simple majority)
- Adding, archiving, or removing subprojects (simple majority)
- Utilizing ORAS/CNCF money for anything CNCF deems "not cheap and easy" (simple majority)

Other decisions may, but do not need to be, called out and put up for decision via creating an [issue][oras-issues-new] at any time and by anyone. By default, any decisions called to a vote will be for a _simple majority_ vote.

## Code of Conduct

This ORAS project has adopted the [CNCF Code of Conduct](https://github.com/cncf/foundation/blob/master/code-of-conduct.md).

## Attributions

* This governance model we created using both the [SPIFFE](https://github.com/spiffe/spire/blob/main/MAINTAINERS.md) and [Helm](https://github.com/helm/community/blob/main/governance/governance.md) governance documents.

## DCO and Licenses

The following licenses and contributor agreements will be used for ORAS projects:

- [Apache 2.0](https://opensource.org/licenses/Apache-2.0) for code
- [Developer Certificate of Origin](https://developercertificate.org/) for new contributions

[lazy-consensus]:     http://communitymgt.wikia.com/wiki/Lazy_consensus
[majority]:           https://en.wikipedia.org/wiki/Majority
[oras-issues-new]:    https://github.com/oras-project/oras/issues/new
[super-majority]:     https://en.wikipedia.org/wiki/Supermajority#Two-thirds_vote
