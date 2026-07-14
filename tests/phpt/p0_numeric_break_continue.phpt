--TEST--
p0 numeric break and continue
--FILE--
<?php
for ($i = 0; $i < 3; $i = $i + 1) {
    for ($j = 0; $j < 3; $j = $j + 1) {
        if ($i == 1 && $j == 0) {
            continue 2;
        }
        if ($i == 2 && $j == 1) {
            break 2;
        }
        echo $i;
        echo $j;
    }
}
echo ':';
for ($k = 0; $k < 2; $k = $k + 1) {
    switch ($k) {
        case 1:
            continue 2;
        default:
            echo $k;
    }
    echo '-';
}
?>
--EXPECT--
00010220:0-
