import shlex


def main() -> None:
    parts = shlex.split("echo 'hello world' --flag=1")
    print(len(parts))
    for p in parts:
        print(p)
    print(shlex.quote("simple"))
    print(shlex.quote("has space"))
    print(shlex.join(["a", "b c", "d"]))


if __name__ == "__main__":
    main()
