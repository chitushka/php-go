--TEST--
Smoke 004: if/elseif/else, switch, while, do-while, for, foreach, alternative syntax
--FILE--
<?php
$x = 5;
if ($x > 10) {
    echo "big\n";
} elseif ($x > 3) {
    echo "medium\n";
} else {
    echo "small\n";
}

switch ($x) {
    case 1:
    case 2:
        echo "low\n";
        break;
    case 5:
        echo "five\n";
        break;
    default:
        echo "other\n";
}

$i = 0;
while ($i < 3) {
    echo "w$i ";
    $i++;
}
echo "\n";

$i = 0;
do {
    echo "d$i ";
    $i++;
} while ($i < 3);
echo "\n";

for ($i = 0; $i < 3; $i++) {
    echo "f$i ";
}
echo "\n";

foreach (array('a', 'b', 'c') as $k => $v) {
    echo "$k=$v ";
}
echo "\n";

if ($x == 5):
    echo "alt-syntax-ok\n";
endif;
--EXPECT--
medium
five
w0 w1 w2 
d0 d1 d2 
f0 f1 f2 
0=a 1=b 2=c 
alt-syntax-ok
