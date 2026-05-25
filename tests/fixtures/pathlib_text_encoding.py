import os
import tempfile
from pathlib import Path


def main() -> None:
    base = tempfile.mkdtemp(prefix="gopy_te_")
    p = Path(os.path.join(base, "data.txt"))

    # CPython honors the encoding kwarg; gopy strings are UTF-8 already
    # so the kwarg is accepted-and-dropped.
    p.write_text("héllo", encoding="utf-8")
    print(p.read_text(encoding="utf-8"))


if __name__ == "__main__":
    main()
