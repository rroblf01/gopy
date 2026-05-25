import os
import tempfile
from pathlib import Path


def main() -> None:
    base = tempfile.mkdtemp(prefix="gopy_pm_")
    src = Path(os.path.join(base, "a.txt"))
    src.write_text("hello")

    # Rename a -> b
    dst = src.rename(os.path.join(base, "b.txt"))
    print(src.exists())
    print(dst.exists())
    print(dst.read_text())

    # Replace b -> c (alias of rename)
    final = dst.replace(os.path.join(base, "c.txt"))
    print(final.read_text())

    # chmod to 0o600; reading after chmod still works.
    final.chmod(0o600)
    print(final.read_text())


if __name__ == "__main__":
    main()
