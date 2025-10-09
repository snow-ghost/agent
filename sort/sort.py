from typing import List, Tuple, Callable
def op_E(a: List[int]) -> None:
    # Swap first and last
    if len(a) >= 2:
        a[0], a[-1] = a[-1], a[0]

def op_L(a: List[int]) -> None:
    # Left cyclic shift
    if a:
        a.append(a.pop(0))

def op_X(a: List[int]) -> None:
    # Swap first two
    if len(a) >= 2:
        a[0], a[1] = a[1], a[0]

def is_sorted(a: List[int]) -> bool:
    return all(a[i] <= a[i+1] for i in range(len(a)-1))

def rotate_left(a: List[int], k: int, ops: List[str]) -> None:
    k %= len(a) if a else 1
    for _ in range(k):
        op_L(a); ops.append('L')

def sort_with_LX(a: List[int]) -> Tuple[List[int], List[str]]:
    """
    Сортирует массив, применяя ТОЛЬКО операции L и X.
    Возвращает (отсортированный массив, список строк с применёнными операциями).
    """
    a = list(a)  # не портим исходный
    n = len(a)
    ops: List[str] = []
    if n <= 1:
        return a, ops

    # Классическая «пузырьковая» схема через вращающееся окно
    for pass_idx in range(n):
        swapped = False
        # За один проход делаем n-1 сравнений, между ними двигаем окно одной L
        for _ in range(n - 1):
            # Сравнить первые два:
            if a[0] > a[1]:
                op_X(a); ops.append('X'); swapped = True
            # Продвинуть окно вправо относительно исходной системы координат:
            op_L(a); ops.append('L')
        # В конце прохода массив сейчас повёрнут влево на (n-1) шагов.
        # Чтобы вернуть в исходную ориентацию, сделаем ещё одну L (итого n L на проход).
        op_L(a); ops.append('L')
        if not swapped:
            break

    return a, ops

def sort_with_LXE(a: List[int]) -> Tuple[List[int], List[str]]:
    """
    Коктейльная сортировка, используя L, X и E.
    Правильная реализация: используем E для оптимизации обратного прохода.
    """
    a = list(a)
    n = len(a)
    ops: List[str] = []
    if n <= 1:
        return a, ops

    left = 0
    right = n - 1
    while left < right:
        swapped = False
        
        # Прямой проход: обычный пузырёк с L
        for i in range(left, right):
            # Подвести пару (i, i+1) к позициям (0, 1)
            rotate_left(a, i, ops)
            if a[0] > a[1]:
                op_X(a); ops.append('X'); swapped = True
        right -= 1
        if not swapped:
            break

        swapped = False
        # Обратный проход: используем E для оптимизации
        for i in range(right, left, -1):
            # Подвести пару (i-1, i) к позициям (0, 1)
            # Выбираем более короткий путь: L (i-1) раз или L (n-i) + E
            cost_direct = i - 1
            cost_reverse = (n - i) + 1  # +1 за E
            
            if cost_reverse < cost_direct and n > 2:
                # Используем обратный путь: L (n-i) + E
                rotate_left(a, n - i, ops)
                op_E(a); ops.append('E')
            else:
                # Используем прямой путь: L (i-1) раз
                rotate_left(a, i - 1, ops)
            
            if a[0] > a[1]:
                op_X(a); ops.append('X'); swapped = True
        left += 1
        if not swapped:
            break

    return a, ops

# Пример проверки
if __name__ == "__main__":
    test_cases = [
        [5, 1, 4, 2, 8, 0, 2],
        [3, 1, 2],
        [1],
        [],
        [2, 1],
        [1, 2, 3, 4, 5],
        [5, 4, 3, 2, 1],
        [1, 1, 1, 1],
        [3, 2, 1, 2, 3]
    ]
    
    for i, data in enumerate(test_cases):
        print(f"\nТест {i+1}: {data}")
        s1, ops1 = sort_with_LX(data)
        s2, ops2 = sort_with_LXE(data)
        print(f"LX:  {s1} ({len(ops1)} операций)")
        print(f"LXE: {s2} ({len(ops2)} операций)")
        print(f"Ожидаемо: {sorted(data)}")
        if s2 != sorted(data):
            print(f"LXE операции: {ops2}")
        assert s1 == sorted(data), f"LX failed on {data}"
        assert s2 == sorted(data), f"LXE failed on {data}"
        print("✓ Оба алгоритма работают корректно")