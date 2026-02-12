# Disclaimer

## No Warranty

compose-diff is provided "as is" without warranty of any kind, express or implied. The authors make no claims about the suitability of this software for any purpose.

## Read-Only Analysis

compose-diff performs **read-only analysis** of Docker Compose files. It does not:

- Execute any Docker commands
- Modify any files
- Connect to Docker daemons
- Deploy or apply configurations

## No Production Claims

compose-diff is designed for **local development and testing workflows**. It makes no guarantees that:

- Detected changes are exhaustive
- "Potential breaking" warnings indicate actual breaking changes
- Stacks will function correctly after changes
- Diff output is suitable for production decision-making

## Informational Only

All output from compose-diff, including severity classifications and "potential breaking" markers, is **informational only**. Users are responsible for:

- Validating changes through proper testing
- Understanding the implications of configuration changes
- Making informed decisions based on their specific requirements

## Liability

In no event shall the authors or copyright holders be liable for any claim, damages, or other liability arising from the use of this software.
