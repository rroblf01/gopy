import argparse


def even_int(s: str) -> int:
    n = int(s)
    if n % 2 != 0:
        raise ValueError("must be even")
    return n


def main() -> None:
    p = argparse.ArgumentParser()
    p.add_argument("--n", type=even_int, default=4)
    p.parse_args(["--n", "12"])
    print(even_int("12"))


if __name__ == "__main__":
    main()
