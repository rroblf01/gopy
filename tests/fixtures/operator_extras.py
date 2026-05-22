import operator


def main() -> None:
    print(operator.truediv(7, 2))
    print(operator.floordiv(7, 2))
    print(operator.mod(7, 2))
    print(operator.neg(5))
    print(operator.abs(-5))
    print(operator.lt(1, 2))
    print(operator.le(2, 2))
    print(operator.eq("a", "a"))
    print(operator.ne(1, 2))
    print(operator.gt(3, 2))
    print(operator.ge(3, 3))
    print(operator.not_(False))
    print(operator.truth(0))
    print(operator.truth(1))


if __name__ == "__main__":
    main()
