--TEST--
function defaults globals static locals and references
--FILE--
<?php
function label($name = 'php', $suffix = '!') {
    return $name . $suffix;
}
echo label();
echo label('go', '?');
echo '|';

$x = 1;
function bump_global() {
    global $x;
    $x = $x + 2;
}
bump_global();
echo $x;
echo '|';

function next_value() {
    static $i = 0;
    $i = $i + 1;
    return $i;
}
echo next_value();
echo next_value();
echo next_value();
echo '|';

function inc(&$value) {
    $value = $value + 1;
}
$n = 4;
inc($n);
echo $n;
--EXPECT--
php!go?|3|123|5
