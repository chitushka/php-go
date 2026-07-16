--TEST--
P2 concrete subclass must implement inherited interface
--FILE--
<?php
interface InheritedNamed {
    public function name();
}
abstract class InheritedAbstractUser implements InheritedNamed {
}
class MissingInheritedUser extends InheritedAbstractUser {
}
?>
--EXPECTF--
class MissingInheritedUser must implement interface method InheritedNamed::name
