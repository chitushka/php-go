--TEST--
P2 concrete subclass inherited interface method must be public
--FILE--
<?php
interface InheritedPublicNamed {
    public function name();
}
abstract class InheritedProtectedBase implements InheritedPublicNamed {
}
class InheritedProtectedUser extends InheritedProtectedBase {
    protected function name() {}
}
?>
--EXPECTF--
class InheritedProtectedUser interface method name must be public
