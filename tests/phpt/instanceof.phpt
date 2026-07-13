--TEST--
instanceof classes and interfaces
--FILE--
<?php
interface Named {}
interface UserNamed extends Named {}
class Base {}
class User extends Base implements UserNamed {}
$user = new User();
echo $user instanceof User;
echo $user instanceof Base;
echo $user instanceof UserNamed;
echo $user instanceof Named;
echo $user instanceof Missing;
--EXPECT--
1111
