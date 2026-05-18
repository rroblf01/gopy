class Animal:
    def __init__(self, name: str) -> None:
        self.name = name


class Dog(Animal):
    def __init__(self, name: str) -> None:
        super().__init__(name)


def main() -> None:
    d: Dog = Dog("rex")
    a: Animal = Animal("kitty")
    # Tuple-of-classes check returns true on first match.
    print(isinstance(d, (Dog, Animal)))
    print(isinstance(a, (Dog, Animal)))
    print(isinstance("hi", (int, str)))
    print(isinstance(5, (int, str)))
    print(isinstance(5.5, (int, str)))
    print(issubclass(Dog, Animal))
    print(issubclass(Animal, Dog))


if __name__ == "__main__":
    main()
