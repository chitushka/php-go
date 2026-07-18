--TEST--
P3 user function missing argument warnings are non-halting by default
--FILE--
<?php
function pair($a, $b) {
    echo $a;
    echo ":";
    echo ($b === null ? 'null' : 'bad');
}
pair('set');
echo "\n";
$fn = function($a, $b) {
    echo $a;
    echo ":";
    echo ($b === null ? 'null' : 'bad');
};
$fn('closure');
echo "\n";
echo 'after';
?>
--EXPECT--
set:null
closure:null
after
