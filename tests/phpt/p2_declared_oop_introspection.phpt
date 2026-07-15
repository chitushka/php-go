--TEST--
P2 declared OOP introspection functions
--FILE--
<?php
interface DeclaredContract {}
interface DeclaredChild extends DeclaredContract {}

class DeclaredBase {}
class DeclaredUser extends DeclaredBase implements DeclaredChild {}

$classes = get_declared_classes();
$interfaces = get_declared_interfaces();

echo in_array('Exception', $classes) ? 'exception' : 'no';
echo ':';
echo in_array('DeclaredBase', $classes) ? 'base' : 'no';
echo ':';
echo in_array('DeclaredUser', $classes) ? 'user' : 'no';
echo ':';
echo in_array('DeclaredContract', $interfaces) ? 'contract' : 'no';
echo ':';
echo in_array('DeclaredChild', $interfaces) ? 'child' : 'no';
--EXPECT--
exception:base:user:contract:child
