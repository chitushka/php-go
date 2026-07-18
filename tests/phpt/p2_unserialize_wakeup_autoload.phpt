--TEST--
P2 unserialize autoloads class before __wakeup
--FILE--
<?php
function __autoload($class) {
    include 'p2_unserialize_wakeup_autoload_lib.php';
}

$obj = unserialize('O:8:"WakeAuto":0:{}');
echo get_class($obj);
echo ':';
echo $obj->ready;
?>
--EXPECT--
WakeAuto:yes
