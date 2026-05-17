def join(sep: str, *parts: str) -> str:
    out: str = ""
    i: int = 0
    for p in parts:
        if i > 0:
            out += sep
        out += str(p)
        i += 1
    return out


def show(**opts: int) -> int:
    total: int = 0
    for k in opts:
        total += int(opts[k])
    return total


def main() -> None:
    print(join("-", "ada", "grace", "alan"))
    print(join(":"))
    print(show(a=1, b=2, c=3))


if __name__ == "__main__":
    main()
