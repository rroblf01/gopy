from typing import final


@final
class Sealed:
    def x(self) -> int:
        return 1


def main() -> None:
    s = Sealed()
    print(s.x())


if __name__ == "__main__":
    main()
