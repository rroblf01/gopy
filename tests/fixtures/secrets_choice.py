import secrets


def main() -> None:
    pool: list[int] = [10, 20, 30, 40, 50]
    pick: int = secrets.choice(pool)
    if pick in pool:
        print("ok")
    else:
        print("bad")

    words: list[str] = ["alpha", "beta", "gamma"]
    w: str = secrets.choice(words)
    if w in words:
        print("ok")
    else:
        print("bad")


if __name__ == "__main__":
    main()
