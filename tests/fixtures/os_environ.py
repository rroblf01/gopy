import os


def main() -> None:
    env: dict[str, str] = os.environ
    home: str = env.get("HOME", "missing")
    print(len(home) > 0)
    print("HOME" in env)
    print("__GOPY_NONEXISTENT__" in env)
    print(os.cpu_count() > 0)
    print(os.path.isabs("/tmp"))
    print(os.path.isabs("relative/path"))


if __name__ == "__main__":
    main()
