from typing import Union


class Dog:
    def __init__(self, name: str) -> None:
        self.name = name

    def sound(self) -> str:
        return f"{self.name}: woof"


class Cat:
    def __init__(self, name: str) -> None:
        self.name = name

    def sound(self) -> str:
        return f"{self.name}: meow"


def describe(p: Union[Dog, Cat]) -> str:
    if isinstance(p, Dog):
        return p.sound()
    if isinstance(p, Cat):
        return p.sound()
    return "unknown"


def stringify(x: Union[int, str]) -> str:
    if isinstance(x, int):
        return f"int={x + 1}"
    if isinstance(x, str):
        return f"str={x.upper()}"
    return "?"


def main() -> None:
    print(describe(Dog("rex")))
    print(describe(Cat("luna")))
    print(stringify(10))
    print(stringify("hello"))


if __name__ == "__main__":
    main()
