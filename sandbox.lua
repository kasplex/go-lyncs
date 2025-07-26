(function()
--[[-code-callbacks-]]
	local _G_RAW = _G
	local _set = function (t)
		local _setmt=setmetatable
		if t==_G_RAW then
			setmetatable=nil
			getmetatable=nil
		end
		local mt = {
			__index=t,
			__newindex=function(_, k, v)
				if t~=_G_RAW then error("variable read-only "..k,2) end
				if k=="session" or k=="state" then t[k]=v; return end
				if fn[k] and t[k]==nil and type(v)=="function" then t[k]=v; return end
				error("variable read-only "..k, 2)
			end
		}
		return _setmt({}, mt)
	end
	table = _set(table)
	string = _set(string)
	math = _set(math)
	bit = _set(bit)
	mpz = _set(mpz)
--[[-code-readonly-list-]]
--[[-code-debug-]]
	_G = _set(_G_RAW)
end)();
