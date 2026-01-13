# Contributing

## Development setup

- Python 3.12+

Create a virtual environment and install dependencies:

- `python -m venv .venv`
- Activate it:
  - Windows (PowerShell): `.\.venv\Scripts\Activate.ps1`
  - macOS/Linux: `source .venv/bin/activate`
- `pip install -r requirements.txt`
- `pip install -r requirements-dev.txt`

## Linting

Run:

- `python -m pylint main.py ssh_utils.py`

## Running

- `python main.py`

## Pull requests

- Keep changes focused and small.
- Prefer UI changes that maintain a clear, guided flow.
- Avoid introducing platform-specific behavior unless necessary.
