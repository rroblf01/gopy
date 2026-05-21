from typing import final, overload, no_type_check, override
from functools import cached_property


@final
class Sealed:
    __slots__ = ("name",)
    __match_args__ = ("name",)

    def __init__(self, name: str) -> None:
        self.name = name

    @final
    def hello(self) -> str:
        return f"hi {self.name}"


class Parent:
    def base(self) -> str:
        return "parent"


class Child(Parent):
    @override
    def base(self) -> str:
        return "child"


@no_type_check
def loose(x: int) -> int:
    return x + 1


@overload
def stringify(x: int) -> str: ...


def stringify(x: int) -> str:
    return str(x)


def main() -> None:
    s = Sealed("ada")
    print(s.hello())
    print(Child().base())
    print(loose(7))
    print(stringify(5))


if __name__ == "__main__":
    main()
