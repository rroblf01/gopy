import os
import tempfile
from pathlib import Path


def main() -> None:
    base = tempfile.mkdtemp(prefix="gopy_rl_")
    base_p = Path(base)

    sub = Path(os.path.join(base, "sub", "leaf.txt"))
    os.makedirs(os.path.dirname(str(sub)))
    sub.write_text("x")

    # is_relative_to + relative_to
    print(sub.is_relative_to(base_p))
    print(sub.is_relative_to(Path("/elsewhere")))
    print(str(sub.relative_to(base_p)))


if __name__ == "__main__":
    main()
