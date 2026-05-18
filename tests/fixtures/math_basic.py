import math


def main() -> None:
    print(math.sqrt(16.0))
    print(math.floor(3.7))
    print(math.ceil(3.2))
    # math.pi rounded to 4 decimals to dodge floating-point noise across
    # runtimes / library versions.
    print(round(math.pi * 10000.0))
    print(math.pow(2.0, 10.0))


if __name__ == "__main__":
    main()
