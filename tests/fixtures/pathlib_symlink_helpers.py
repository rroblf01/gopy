import os
import tempfile
from pathlib import Path


def main() -> None:
    base = tempfile.mkdtemp(prefix="gopy_sl_")
    real = Path(os.path.join(base, "real.txt"))
    real.write_text("data")

    # Not a symlink — plain regular file.
    print(real.is_symlink())

    # Two paths to the same file.
    other = Path(os.path.join(base, "real.txt"))
    print(real.samefile(other))
    print(real.samefile(str(other)))

    # as_posix replaces OS separators with '/'.
    print(Path("a/b/c.txt").as_posix())


if __name__ == "__main__":
    main()
