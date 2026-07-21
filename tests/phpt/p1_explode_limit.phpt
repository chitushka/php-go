--TEST--
p1 explode() supports the optional $limit argument (BUG-8)
--FILE--
<?php
print_r(explode(",", "a,b,c,d", 2));
print_r(explode(",", "a,b,c,d", -1));
print_r(explode(",", "a,b,c,d", -6));
print_r(explode(",", "a,b,c,d", 0));
print_r(explode(",", "a,b,c,d", 100));
?>
--EXPECT--
Array
(
    [0] => a
    [1] => b,c,d
)
Array
(
    [0] => a
    [1] => b
    [2] => c
)
Array
(
)
Array
(
    [0] => a,b,c,d
)
Array
(
    [0] => a
    [1] => b
    [2] => c
    [3] => d
)
