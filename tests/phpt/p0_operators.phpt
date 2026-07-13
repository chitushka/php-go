--TEST--
p0 word bitwise shift ternary operators
--FILE--
<?php
function fail() {
    return 1 / 0;
}
if (0 and fail()) {
    echo 'bad';
} else {
    echo 'and';
}
if (1 or fail()) {
    echo 'or';
}
echo ':';
echo 6 & 3;
echo ':';
echo 4 | 1;
echo ':';
echo 5 ^ 3;
echo ':';
echo ~0;
echo ':';
echo 1 << 3;
echo ':';
echo 8 >> 1;
echo ':';
echo +5;
echo ':';
echo 0 ? 'bad' : 'ternary';
echo ':';
echo 'fallback' ?: 'bad';
echo ':';
if (1 xor 0) {
    echo 'xor';
}
--EXPECT--
andor:2:5:6:-1:8:4:5:ternary:fallback:xor
