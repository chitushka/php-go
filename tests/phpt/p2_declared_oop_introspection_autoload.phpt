--TEST--
P2 declared OOP introspection includes autoloaded classes and interfaces
--FILE--
<?php
function __autoload($class) {
    include 'p2_declared_oop_introspection_autoload_lib.php';
}

$classes = get_declared_classes();
$interfaces = get_declared_interfaces();

echo in_array('DeclaredAutoloaded', $classes) ? 'loaded' : 'missing';
echo ':';
echo in_array('DeclaredAutoloadContract', $interfaces) ? 'loaded' : 'missing';
echo "\n";

echo class_exists('DeclaredAutoloaded') ? 'class' : 'no';
echo "\n";

$classes = get_declared_classes();
$interfaces = get_declared_interfaces();

echo in_array('DeclaredAutoloaded', $classes) ? 'loaded' : 'missing';
echo ':';
echo in_array('DeclaredAutoloadContract', $interfaces) ? 'loaded' : 'missing';
--EXPECT--
missing:missing
class
loaded:loaded
