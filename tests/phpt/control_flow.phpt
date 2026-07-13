--TEST--
for do while switch break continue
--FILE--
<?php
for ($i = 0; $i < 6; $i = $i + 1) {
    if ($i == 1) {
        continue;
    }
    if ($i == 4) {
        break;
    }
    echo $i;
}
echo '|';

$j = 0;
do {
    echo $j;
    $j = $j + 1;
} while ($j < 3);
echo '|';

foreach (array(1, 2, 3) as $v) {
    if ($v == 2) {
        continue;
    }
    echo $v;
}
echo '|';

switch (2) {
    case 1:
        echo 'a';
        break;
    case 2:
        echo 'b';
    case 3:
        echo 'c';
        break;
    default:
        echo 'd';
}
switch (9) {
    case 1:
        echo 'x';
        break;
    default:
        echo 'z';
}
--EXPECT--
023|012|13|bcz
