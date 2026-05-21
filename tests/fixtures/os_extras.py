import os
import shutil


def main() -> None:
    sh: str = shutil.which("sh")
    print(len(sh) > 0 or sh == "")
    missing = shutil.which("__gopy_nonexistent_cmd__")
    print(missing in (None, ""))
    size: int = os.path.getsize("/")
    print(size >= 0)
    mtime: float = os.path.getmtime("/")
    print(mtime > 0)


if __name__ == "__main__":
    main()
